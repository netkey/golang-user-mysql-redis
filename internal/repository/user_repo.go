package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/netkey/golang-user-mysql-redis/internal/model"
	"github.com/redis/go-redis/v9"
)

type UserRepository interface {
	Create(ctx context.Context, u *model.User) error
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id int) (*model.User, error)
	UpdateProfile(ctx context.Context, id int, nickname string, age int, avatar string) error

	// 缓存操作
	GetCache(ctx context.Context, id int) (*model.User, error)
	SetCache(ctx context.Context, user *model.User) error
	DeleteCache(ctx context.Context, id int) error

	// 好友操作
	AddFriend(ctx context.Context, userID, friendID int) error
	GetFriends(ctx context.Context, userID int) ([]model.User, error)
}

type userRepo struct {
	db    *sql.DB
	redis *redis.Client
}

func NewUserRepository(db *sql.DB, rdb *redis.Client) UserRepository {
	return &userRepo{db: db, redis: rdb}
}

// --- 数据库操作 (纯标准库 *sql.DB 实现) ---

func (r *userRepo) Create(ctx context.Context, u *model.User) error {
	// 使用原生 SQL 和 ? 占位符，注意参数顺序必须与 SQL 对应
	query := `INSERT INTO users (name, nickname, email, password, age, gender, avatar, status, created_at, updated_at) 
              VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`

	_, err := r.db.ExecContext(ctx, query,
		u.Name, u.Nickname, u.Email, u.Password, u.Age, u.Gender, u.Avatar, u.Status,
	)
	return err
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	query := "SELECT id, name, nickname, email, password, age, gender, avatar, status FROM users WHERE email = ? LIMIT 1"

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID, &u.Name, &u.Nickname, &u.Email, &u.Password, &u.Age, &u.Gender, &u.Avatar, &u.Status,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &u, err
}

func (r *userRepo) GetByID(ctx context.Context, id int) (*model.User, error) {
	var u model.User
	query := "SELECT id, name, nickname, email, age, gender, avatar, status FROM users WHERE id = ?"

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Name, &u.Nickname, &u.Email, &u.Age, &u.Gender, &u.Avatar, &u.Status,
	)
	return &u, err
}

func (r *userRepo) UpdateProfile(ctx context.Context, id int, nickname string, age int, avatar string) error {
	query := `UPDATE users SET nickname = ?, age = ?, avatar = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, nickname, age, avatar, id)
	return err
}

// --- 好友操作 ---

func (r *userRepo) AddFriend(ctx context.Context, userID, friendID int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `INSERT IGNORE INTO friends (user_id, friend_id, status) VALUES (?, ?, 2)`
	if _, err := tx.ExecContext(ctx, query, userID, friendID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, query, friendID, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *userRepo) GetFriends(ctx context.Context, userID int) ([]model.User, error) {
	query := `
		SELECT u.id, u.name, u.nickname, u.email, u.avatar 
		FROM users u
		INNER JOIN friends f ON u.id = f.friend_id
		WHERE f.user_id = ? AND f.status = 2`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var friends []model.User
	for rows.Next() {
		var f model.User
		if err := rows.Scan(&f.ID, &f.Name, &f.Nickname, &f.Email, &f.Avatar); err != nil {
			return nil, err
		}
		friends = append(friends, f)
	}
	return friends, nil
}

// --- 缓存操作 (保持原样) ---

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
