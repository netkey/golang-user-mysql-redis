package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/netkey/golang-user-mysql-redis/internal/service"
)

// Response 统一的 JSON 返回结构
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// Register 注册接口 (POST /api/v1/register)
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Nickname string `json:"nickname"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendJSON(w, http.StatusBadRequest, "无效的请求参数", nil)
		return
	}

	// 调用 Service 层注册逻辑（包含密码哈希）
	err := h.svc.Register(r.Context(), req.Name, req.Nickname, req.Email, req.Password)
	if err != nil {
		h.sendJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	h.sendJSON(w, http.StatusOK, "注册成功", nil)
}

// Login 登录接口 (POST /api/v1/login)
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendJSON(w, http.StatusBadRequest, "无效的请求参数", nil)
		return
	}

	token, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		h.sendJSON(w, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	h.sendJSON(w, http.StatusOK, "登录成功", map[string]string{"token": token})
}

// GetProfile 获取个人资料-个人中心 (GET /api/v1/me)
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// 1. 获取用户 ID
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		h.sendJSON(w, http.StatusUnauthorized, "请先登录", nil)
		return
	}

	// 2. 调用 Service 获取用户信息（此时已包含 Redis 缓存和 Singleflight 保护）
	user, err := h.svc.GetUser(r.Context(), userID)
	if err != nil {
		h.sendJSON(w, http.StatusInternalServerError, "获取资料失败", nil)
		return
	}

	// 3. 生成强 ETag 标识 (基于用户 ID 和 更新时间)
	// 如果用户资料修改了，UpdatedAt 会变，ETag 就会失效，CDN 就会回源更新
	etag := fmt.Sprintf(`W/"user-%d-%d"`, user.ID, user.UpdatedAt.Unix())

	// 4. 检查浏览器/CDN 传来的 If-None-Match 头部
	// 如果一致，说明客户端或 CDN 的数据还是最新的
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified) // 直接返回 304，不走下面的 JSON 序列化和网络传输
		return
	}

	// 5. 设置缓存控制头
	// private: 必须，防止 CDN 错误地分发给其他用户
	// max-age: 浏览器和 CDN 的缓存时长
	// Vary: 告知 CDN 根据 Token 不同来区分缓存副本
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "private, max-age=60, stale-while-revalidate=30")
	w.Header().Set("Vary", "Authorization")

	// 6. 正常返回数据
	h.sendJSON(w, http.StatusOK, "success", user)
}

// GetPublicProfile 获取公开的用户资料 (GET /api/v1/user/public/:id)
func (h *UserHandler) GetPublicProfile(w http.ResponseWriter, r *http.Request) {
	// 1. 解析要查看的目标用户 ID (从 URL 参数获取)
	idStr := r.URL.Query().Get("id")
	targetUserID, _ := strconv.Atoi(idStr)

	// 2. 调用 Service 获取公开信息
	// 注意：Service 内部依然有 Redis 缓存和 Singleflight 保护
	user, err := h.svc.GetUser(r.Context(), targetUserID)
	if err != nil || user == nil {
		h.sendJSON(w, http.StatusNotFound, "用户不存在", nil)
		return
	}

	// 3. 构造公开资料的脱敏模型 (安全第一，不返回 email, status 等)
	publicInfo := map[string]interface{}{
		"id":       user.ID,
		"name":     user.Name,
		"nickname": user.Nickname,
		"avatar":   user.Avatar,
		"age":      user.Age,
	}

	// 4. 计算指纹 (ETag)
	etag := fmt.Sprintf(`W/"pub-%d-%d"`, user.ID, user.UpdatedAt.Unix())
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// 5. 设置缓存头：允许 CDN 共享缓存
	w.Header().Set("ETag", etag)
	// public: 关键！允许所有缓存服务器缓存
	// s-maxage: 专门给 CDN 等代理服务器看的，可以设长一点
	// max-age: 给浏览器看的，可以短一点
	w.Header().Set("Cache-Control", "public, max-age=60, s-maxage=300, stale-while-revalidate=60")

	// 注意：这里不要设置 Vary: Authorization，否则 CDN 无法共享缓存
	w.Header().Set("Vary", "Accept-Encoding")

	h.sendJSON(w, http.StatusOK, "success", publicInfo)
}

// UpdateProfile 更新资料 (POST /api/v1/profile/update)
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value("userID").(int)

	var req struct {
		Nickname string `json:"nickname"`
		Age      int    `json:"age"`
		Avatar   string `json:"avatar"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendJSON(w, http.StatusBadRequest, "参数错误", nil)
		return
	}

	err := h.svc.UpdateMyProfile(r.Context(), userID, req.Nickname, req.Age, req.Avatar)
	if err != nil {
		h.sendJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	h.sendJSON(w, http.StatusOK, "资料更新成功", nil)
}

// AddFriend 添加好友 (POST /api/v1/friend/add)
func (h *UserHandler) AddFriend(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value("userID").(int)

	var req struct {
		FriendID int `json:"friend_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if err := h.svc.AddFriend(r.Context(), userID, req.FriendID); err != nil {
		h.sendJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	h.sendJSON(w, http.StatusOK, "好友申请已发送/已添加", nil)
}

// ListFriends 好友列表 (GET /api/v1/friends)
func (h *UserHandler) ListFriends(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value("userID").(int)

	friends, err := h.svc.ListFriends(r.Context(), userID)
	if err != nil {
		h.sendJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	h.sendJSON(w, http.StatusOK, "success", friends)
}

// sendJSON 内部辅助方法，减少重复代码
func (h *UserHandler) sendJSON(w http.ResponseWriter, code int, msg string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(Response{
		Code: code,
		Msg:  msg,
		Data: data,
	})
}
