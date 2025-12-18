package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/netkey/golang-user-mysql-redis/internal/model"
	"github.com/redis/go-redis/v9"
	"time"
)

type UserRepository interface {
	GetByID(ctx context.Context, id int) (*model.User, error)
	GetCache(ctx context.Context, id int) (*model.User, error)
	SetCache(ctx context.Context, user *model.User) error
	DeleteCache(ctx context.Context, id int) error
	Update(ctx context.Context, id int, name string) error
}

type userRepo struct {
	db    *sql.DB
	redis *redis.Client
}

func NewUserRepository(db *sql.DB, rdb *redis.Client) UserRepository {
	return &userRepo{db: db, redis: rdb}
}

func (r *userRepo) GetCache(ctx context.Context, id int) (*model.User, error) {
	val, err := r.redis.Get(ctx, fmt.Sprintf("user:%d", id)).Result()
	if err != nil {
		return nil, err
	}
	var user model.User
	json.Unmarshal([]byte(val), &user)
	return &user, nil
}

func (r *userRepo) SetCache(ctx context.Context, user *model.User) error {
	data, _ := json.Marshal(user)
	return r.redis.Set(ctx, fmt.Sprintf("user:%d", user.ID), data, 15*time.Minute).Err()
}

func (r *userRepo) DeleteCache(ctx context.Context, id int) error {
	return r.redis.Del(ctx, fmt.Sprintf("user:%d", id)).Err()
}

func (r *userRepo) GetByID(ctx context.Context, id int) (*model.User, error) {
	var u model.User
	err := r.db.QueryRowContext(ctx, "SELECT id, name, email FROM users WHERE id = ?", id).
		Scan(&u.ID, &u.Name, &u.Email)
	return &u, err
}

func (r *userRepo) Update(ctx context.Context, id int, name string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE users SET name = ? WHERE id = ?", name, id)
	return err
}
