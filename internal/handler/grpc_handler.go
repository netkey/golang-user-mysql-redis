package handler

import (
	"context"
	"your_project/api/proto/pb"
	"your_project/internal/service"
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
