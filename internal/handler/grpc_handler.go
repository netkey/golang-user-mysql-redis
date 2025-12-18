package handler

import (
	"context"
	"github.com/netkey/golang-user-mysql-redis/internal/service"
	"github.com/netkey/golang-user-mysql-redis/pkg/pb"
)

type UserGRPCHandler struct {
	pb.UnimplementedUserServiceServer
	svc *service.UserService
}

func NewUserGRPCHandler(svc *service.UserService) *UserGRPCHandler {
	return &UserGRPCHandler{svc: svc}
}

func (h *UserGRPCHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	user, err := h.svc.GetUser(ctx, int(req.Id))
	if err != nil {
		return nil, err
	}
	return &pb.UserResponse{
		Id:    int32(user.ID),
		Name:  user.Name,
		Email: user.Email,
	}, nil
}
