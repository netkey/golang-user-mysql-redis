package middleware

import (
	"context"
	"time"

	"github.com/netkey/golang-user-mysql-redis/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// 1. GrpcLoggingInterceptor: 结构化记录每一次 RPC 调用
func GrpcLoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()

	// 执行业务逻辑
	resp, err := handler(ctx, req)

	// 记录执行时间、方法名和错误状态
	duration := time.Since(start)
	st, _ := status.FromError(err)

	logger.Log.Info("gRPC Request",
		zap.String("method", info.FullMethod),
		zap.Duration("duration", duration),
		zap.String("code", st.Code().String()),
		zap.Error(err),
	)

	return resp, err
}

// 2. GrpcAuthInterceptor: 简单的 gRPC 鉴权 (从 Metadata 中提取 token)
func GrpcAuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// 获取 gRPC 元数据 (类似 HTTP Header)
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "metadata is not provided")
	}

	tokens := md.Get("authorization")
	if len(tokens) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "authorization token is not provided")
	}

	// 这里可以添加 JWT 校验逻辑 (类似 HTTP 中间件)
	// token := tokens[0]
	// ... 校验 ...

	return handler(ctx, req)
}

// 3. GrpcRecoveryInterceptor: 防止单个 Panic 导致整个 Server 崩溃
func GrpcRecoveryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Log.Error("gRPC Panic Recovered", zap.Any("panic", r))
			err = status.Errorf(codes.Internal, "Internal server error")
		}
	}()
	return handler(ctx, req)
}
