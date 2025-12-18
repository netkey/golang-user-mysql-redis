package service

import (
	"context"
	"fmt"
	"golang.org/x/sync/singleflight"
	"your_project/internal/model"
	"your_project/internal/repository"
)

type UserService struct {
	repo repository.UserRepository
	sf   singleflight.Group
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetUser(ctx context.Context, id int) (*model.User, error) {
	// 1. 查缓存
	user, err := s.repo.GetCache(ctx, id)
	if err == nil {
		return user, nil
	}

	// 2. Singleflight 防击穿
	key := fmt.Sprintf("get_user_%d", id)
	v, err, _ := s.sf.Do(key, func() (interface{}, error) {
		u, dbErr := s.repo.GetByID(ctx, id)
		if dbErr != nil {
			return nil, dbErr
		}
		_ = s.repo.SetCache(ctx, u)
		return u, nil
	})

	if err != nil {
		return nil, err
	}
	return v.(*model.User), nil
}

func (s *UserService) UpdateUser(ctx context.Context, id int, name string) (*model.User, error) {
	if err := s.repo.Update(ctx, id, name); err != nil {
		return nil, err
	}
	_ = s.repo.DeleteCache(ctx, id)
	return s.GetUser(ctx, id)
}
