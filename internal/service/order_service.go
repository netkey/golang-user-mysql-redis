package service

import (
	"context"
	"fmt"
	"github.com/netkey/golang-user-mysql-redis/pkg/discovery"
	"github.com/netkey/golang-user-mysql-redis/pkg/logger"
	"github.com/netkey/golang-user-mysql-redis/pkg/pb"
)

type OrderService struct {
	userClient pb.UserServiceClient // gRPC 客户端
}

func NewOrderService(etcdEndpoints []string) (*OrderService, error) {
	// 1. 通过 Etcd 发现并建立连接
	conn, err := discovery.GetGRPCClient(etcdEndpoints, "user-service")
	if err != nil {
		return nil, err
	}

	// 2. 初始化客户端
	return &OrderService{
		userClient: pb.NewUserServiceClient(conn),
	}, nil
}

func (s *OrderService) CreateOrder(ctx context.Context, userID int32) error {
	// 3. 跨服务调用 User Service 的 gRPC 接口
	user, err := s.userClient.GetUserByID(ctx, &pb.GetUserRequest{Id: userID})
	if err != nil {
		logger.Log.Error("调用 User 模块失败", zap.Error(err))
		return fmt.Errorf("failed to fetch user: %v", err)
	}

	fmt.Printf("为用户 %s 下单成功\n", user.Name)
	return nil
}
