package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// 基础库
	"github.com/netkey/golang-user-mysql-redis/internal/config"
	"github.com/netkey/golang-user-mysql-redis/internal/handler"
	"github.com/netkey/golang-user-mysql-redis/internal/middleware"
	"github.com/netkey/golang-user-mysql-redis/internal/repository"
	"github.com/netkey/golang-user-mysql-redis/internal/service"
	"github.com/netkey/golang-user-mysql-redis/pkg/database"
	"github.com/netkey/golang-user-mysql-redis/pkg/discovery"
	"github.com/netkey/golang-user-mysql-redis/pkg/logger"
	"github.com/netkey/golang-user-mysql-redis/pkg/pb"

	// 外部依赖
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	// 1. 初始化结构化日志 (Zap)
	logger.InitLogger()
	defer logger.Log.Sync()

	// 2. 加载配置文件 (Viper)
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		logger.Log.Fatal("配置文件加载失败", zap.Error(err))
	}

	// 3. 初始化持久化层 (MySQL + Redis 并配置连接池)
	// 注意：此处 repository 逻辑需要 db 为 *sql.DB
	db, err := database.NewMySQL(cfg.MySQL.DSN)
	if err != nil {
		logger.Log.Fatal("MySQL 连接失败", zap.Error(err))
	}
	db.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)
	defer db.Close()

	rdb, err := database.NewRedis(cfg.Redis)
	if err != nil {
		logger.Log.Fatal("Redis 连接失败", zap.Error(err))
	}
	defer rdb.Close()

	// 4. 组装依赖注入 (DI)
	// Repo -> Service -> Handler
	userRepo := repository.NewUserRepository(db, rdb)
	userSvc := service.NewUserService(userRepo, cfg) // 传入 cfg 供 JWT 使用
	userHandler := handler.NewUserHandler(userSvc)

	// 5. 配置 HTTP 服务器 (REST API + Metrics)
	mux := http.NewServeMux()

	// --- A. 公开接口 (无需鉴权) ---
	mux.HandleFunc("/api/v1/register", userHandler.Register)
	mux.HandleFunc("/api/v1/login", userHandler.Login)
	mux.Handle("/metrics", promhttp.Handler()) // Prometheus 采集接口

	// --- B. 私有接口 (应用 JWT 鉴权中间件) ---
	// 我们可以封装一个简单的路由装饰器或使用第三方路由库，这里使用标准库演示
	auth := middleware.AuthMiddleware(cfg.JWT.Secret)

	mux.Handle("/api/v1/me", auth(http.HandlerFunc(userHandler.GetProfile)))
	mux.Handle("/api/v1/profile/update", auth(http.HandlerFunc(userHandler.UpdateProfile)))
	mux.Handle("/api/v1/friends", auth(http.HandlerFunc(userHandler.ListFriends)))
	mux.Handle("/api/v1/friend/add", auth(http.HandlerFunc(userHandler.AddFriend)))

	// 全局中间件应用 (如 Prometheus Metrics)
	var finalHandler http.Handler = mux
	finalHandler = middleware.MetricsMiddleware(finalHandler)

	httpSrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.HttpPort),
		Handler: finalHandler,
	}

	// 6. 配置 gRPC 服务器 (用于内部服务间通信)
	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.GrpcRecoveryInterceptor,
			middleware.GrpcLoggingInterceptor,
			middleware.GrpcAuthInterceptor,
		),
	)

	// 注册 gRPC 服务实现
	userGRPCHandler := handler.NewUserGRPCHandler(userSvc)
	pb.RegisterUserServiceServer(grpcSrv, userGRPCHandler)

	// 7. 初始化 Etcd 服务注册
	reg, err := discovery.NewRegister(cfg.Etcd.Endpoints)
	if err != nil {
		logger.Log.Fatal("Etcd 初始化失败", zap.Error(err))
	}

	// 8. 启动服务

	// 启动 gRPC (需先监听端口)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GrpcPort))
	if err != nil {
		logger.Log.Fatal("gRPC 端口监听失败", zap.Error(err))
	}

	go func() {
		logger.Log.Info("gRPC Server 正在启动", zap.Int("port", cfg.Server.GrpcPort))
		if err := grpcSrv.Serve(lis); err != nil {
			logger.Log.Error("gRPC Server 停止运行", zap.Error(err))
		}
	}()

	// 启动 HTTP (REST API)
	go func() {
		logger.Log.Info("HTTP Server 正在启动", zap.Int("port", cfg.Server.HttpPort))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("HTTP Server 启动失败", zap.Error(err))
		}
	}()

	// 9. 服务注册到 Etcd
	grpcAddr := fmt.Sprintf("%s:%d", cfg.Server.InternalIP, cfg.Server.GrpcPort)
	go func() {
		err := reg.RegisterService(context.Background(), "user-service", grpcAddr, 10)
		if err != nil {
			logger.Log.Error("服务注册 Etcd 失败", zap.Error(err))
		}
	}()

	// 10. 优雅关闭 (Graceful Shutdown)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	logger.Log.Info("接收到退出信号", zap.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 按照顺序关闭
	reg.Stop()
	grpcSrv.GracefulStop()
	if err := httpSrv.Shutdown(ctx); err != nil {
		logger.Log.Error("HTTP Server 强制关闭", zap.Error(err))
	}

	logger.Log.Info("所有服务已安全退出")
}
