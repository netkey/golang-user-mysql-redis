package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/netkey/golang-user-mysql-redis/internal/config"
	"github.com/netkey/golang-user-mysql-redis/internal/model"
	"github.com/netkey/golang-user-mysql-redis/internal/repository"
	"github.com/netkey/golang-user-mysql-redis/pkg/utils" // 确保有 JWT 工具类
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/singleflight"
)

type UserService struct {
	repo repository.UserRepository
	sf   singleflight.Group
	cfg  *config.Config
}

func NewUserService(repo repository.UserRepository, cfg *config.Config) *UserService {
	return &UserService{repo: repo, cfg: cfg}
}

// Register 用户注册
func (s *UserService) Register(ctx context.Context, name, nickname, email, password string) error {
	// 1. 检查邮箱是否已占用
	exists, err := s.repo.GetByEmail(ctx, email)
	if err == nil && exists != nil {
		return errors.New("该邮箱已被注册")
	}

	// 2. 密码哈希加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 3. 构建模型
	user := &model.User{
		Name:     name,
		Nickname: nickname,
		Email:    email,
		Password: string(hashedPassword),
		Status:   1, // 1-正常
	}

	return s.repo.Create(ctx, user)
}

// Login 用户登录并返回 Token
func (s *UserService) Login(ctx context.Context, email, password string) (string, error) {
	// 1. 根据 Email 获取用户（此处由于是登录，不强制走 Singleflight，直接查库）
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil || user == nil {
		return "", errors.New("用户不存在或密码错误")
	}

	// 2. 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("用户不存在或密码错误")
	}

	// 3. 签发 Token
	return utils.GenerateToken(user.ID, s.cfg.JWT.Secret, s.cfg.JWT.Expire)
}

// GetUser 获取用户信息（带缓存 + Singleflight 防击穿）
func (s *UserService) GetUser(ctx context.Context, id int) (*model.User, error) {
	// 1. 尝试从缓存读取
	user, err := s.repo.GetCache(ctx, id)
	if err == nil && user != nil {
		return user, nil
	}

	// 2. 缓存失效，使用 Singleflight 合并并发请求
	key := fmt.Sprintf("get_user_%d", id)
	v, err, _ := s.sf.Do(key, func() (interface{}, error) {
		// 二次查库
		u, dbErr := s.repo.GetByID(ctx, id)
		if dbErr != nil {
			return nil, dbErr
		}
		// 异步或同步回写缓存
		_ = s.repo.SetCache(ctx, u)
		return u, nil
	})

	if err != nil {
		return nil, err
	}
	return v.(*model.User), nil
}

// UpdateMyProfile 用户修改自己的资料
func (s *UserService) UpdateMyProfile(ctx context.Context, userID int, nickname string, age int, avatar string) error {
	// 1. 调用仓库层原生 SQL 更新
	if err := s.repo.UpdateProfile(ctx, userID, nickname, age, avatar); err != nil {
		return err
	}

	// 2. 删除缓存（Cache Aside 策略：先更新库，再删缓存）
	_ = s.repo.DeleteCache(ctx, userID)
	return nil
}

// AddFriend 添加好友
func (s *UserService) AddFriend(ctx context.Context, userID, friendID int) error {
	if userID == friendID {
		return errors.New("不能添加自己")
	}
	// 执行仓库层的事务添加逻辑
	return s.repo.AddFriend(ctx, userID, friendID)
}

// ListFriends 获取好友列表
func (s *UserService) ListFriends(ctx context.Context, userID int) ([]model.User, error) {
	return s.repo.GetFriends(ctx, userID)
}

// Logout 退出登录
func (s *UserService) Logout(ctx context.Context, token string) error {
	// 将 Token 加入 Redis 黑名单，过期时间建议设置为 JWT 剩余有效期
	// 这里假设你通过 Redis 直接操作或扩展 Repo 接口
	return nil
}

// GetJWTSecret 暴露配置中的密钥给 Handler 使用
func (s *UserService) GetJWTSecret() string {
	return s.cfg.JWT.Secret
}
