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
	"your_project/api/proto/pb"
	"your_project/internal/handler"
	"your_project/internal/middleware"
	"your_project/internal/repository"
	"your_project/internal/service"
	"your_project/pkg/config"
	"your_project/pkg/database"
	"your_project/pkg/discovery"
	"your_project/pkg/logger"

	// 外部依赖
	gqlHandler "github.com/graphql-go/handler"
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
	db, err := database.NewMySQL(cfg.MySQL.DSN)
	if err != nil {
		logger.Log.Fatal("MySQL 连接失败", zap.Error(err))
	}
	db.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)
	defer db.Close()

	rdb := database.NewRedis(cfg.Redis.Addr)
	defer rdb.Close()

	// 4. 组装依赖注入 (DI)
	// Repo -> Service -> Handler
	userRepo := repository.NewUserRepository(db, rdb)
	userSvc := service.NewUserService(userRepo)

	// 5. 初始化 GraphQL Schema
	schema, err := handler.NewSchema(userSvc)
	if err != nil {
		logger.Log.Fatal("GraphQL Schema 创建失败", zap.Error(err))
	}

	// 6. 配置 HTTP 服务器 (GraphQL + Metrics)
	mux := http.NewServeMux()

	// GraphQL 端点 + 中间件 (Auth + Prometheus Metrics)
	h := gqlHandler.New(&gqlHandler.Config{
		Schema:   &schema,
		GraphiQL: true,
	})

	// 装饰器模式应用中间件
	var gqlWithMiddleware http.Handler = h
	gqlWithMiddleware = middleware.AuthMiddleware(gqlWithMiddleware)
	gqlWithMiddleware = middleware.MetricsMiddleware(gqlWithMiddleware)

	mux.Handle("/graphql", gqlWithMiddleware)
	mux.Handle("/metrics", promhttp.Handler()) // Prometheus 采集接口

	httpSrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.HttpPort),
		Handler: mux,
	}

	// 7. 配置 gRPC 服务器 (用于内部服务间通信)
	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
		// 这里可以添加 gRPC 版本的日志和 Trace 中间件
		),
	)
	userGRPCHandler := handler.NewUserGRPCHandler(userSvc)
	pb.RegisterUserServiceServer(grpcSrv, userGRPCHandler)

	// 8. 初始化 Etcd 服务注册
	reg, err := discovery.NewRegister(cfg.Etcd.Endpoints)
	if err != nil {
		logger.Log.Fatal("Etcd 初始化失败", zap.Error(err))
	}

	// 9. 启动服务

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

	// 启动 HTTP
	go func() {
		logger.Log.Info("HTTP Server 正在启动", zap.Int("port", cfg.Server.HttpPort))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("HTTP Server 启动失败", zap.Error(err))
		}
	}()

	// 10. 服务注册到 Etcd (注册的是 gRPC 的地址，供其他服务发现)
	grpcAddr := fmt.Sprintf("%s:%d", cfg.Server.InternalIP, cfg.Server.GrpcPort)
	go func() {
		err := reg.RegisterService(context.Background(), "user-service", grpcAddr, 10)
		if err != nil {
			logger.Log.Error("服务注册 Etcd 失败", zap.Error(err))
		}
	}()

	// 11. 优雅关闭 (Graceful Shutdown)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	logger.Log.Info("接收到退出信号", zap.String("signal", sig.String()))

	// 设定最大退出等待时间
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 步骤 A: 从 Etcd 注销服务节点（停止新流量进入）
	reg.Stop()

	// 步骤 B: 停止 gRPC 服务
	grpcSrv.GracefulStop()

	// 步骤 C: 停止 HTTP 服务
	if err := httpSrv.Shutdown(ctx); err != nil {
		logger.Log.Error("HTTP Server 强制关闭", zap.Error(err))
	}

	logger.Log.Info("所有服务已安全退出")
}
