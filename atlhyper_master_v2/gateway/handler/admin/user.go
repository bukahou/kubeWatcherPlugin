// atlhyper_master_v2/gateway/handler/user.go
// 用户管理 Handler
package admin

import (
	"encoding/json"
	"net/http"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/gateway/middleware"
	"AtlHyper/common/logger"

	"golang.org/x/crypto/bcrypt"
)

var log = logger.Module("UserHandler")

// UserHandler 用户管理 Handler
type UserHandler struct {
	userRepo database.UserRepository
}

// NewUserHandler 创建 UserHandler
func NewUserHandler(userRepo database.UserRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

// ==================== 请求/响应结构 ====================

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}

// UserInfo 用户信息（不含密码）
type UserInfo struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Role        int    `json:"role"`
}

// UserDTO 用户详情（用于列表）
type UserDTO struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Role        int    `json:"role"`
	Status      int    `json:"status"` // 1=Active, 0=Disabled
	CreatedAt   string `json:"createdAt"`
}

// RegisterRequest 注册请求（仅 Admin 可调用）
type RegisterRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Role        int    `json:"role"` // 1=Viewer, 2=Operator, 3=Admin
}

// UpdateRoleRequest 更新角色请求
type UpdateRoleRequest struct {
	UserID int64 `json:"userId"`
	Role   int   `json:"role"`
}

// DeleteUserRequest 删除用户请求
type DeleteUserRequest struct {
	UserID int64 `json:"userId"`
}

// UpdateStatusRequest 更新用户状态请求
type UpdateStatusRequest struct {
	UserID int64 `json:"userId"`
	Status int   `json:"status"` // 1=Active, 0=Disabled
}

// ==================== Handler 方法 ====================

// Login 用户登录
// POST /api/v2/user/login
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "请求参数无效")
		return
	}

	if req.Username == "" || req.Password == "" {
		handler.WriteError(w, http.StatusBadRequest, "用户名和密码不能为空")
		return
	}

	// 查找用户
	user, err := h.userRepo.GetByUsername(r.Context(), req.Username)
	if err != nil {
		log.Error("查询用户失败", "err", err)
		handler.WriteError(w, http.StatusInternalServerError, "查询用户失败: "+err.Error())
		return
	}
	if user == nil {
		handler.WriteError(w, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		handler.WriteError(w, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	// 检查用户状态
	if user.Status != 1 {
		handler.WriteError(w, http.StatusForbidden, "账号已被禁用")
		return
	}

	// 生成 Token
	token, err := middleware.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "生成 Token 失败")
		return
	}

	// 更新登录时间
	clientIP := r.RemoteAddr
	if xForwardedFor := r.Header.Get("X-Forwarded-For"); xForwardedFor != "" {
		clientIP = xForwardedFor
	}
	_ = h.userRepo.UpdateLastLogin(r.Context(), user.ID, clientIP)

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "登录成功",
		"data": LoginResponse{
			Token: token,
			User: UserInfo{
				ID:          user.ID,
				Username:    user.Username,
				DisplayName: user.DisplayName,
				Email:       user.Email,
				Role:        user.Role,
			},
		},
	})
}

// Register 用户注册
// POST /api/v2/user/register
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "请求参数无效")
		return
	}

	if req.Username == "" || req.Password == "" {
		handler.WriteError(w, http.StatusBadRequest, "用户名和密码不能为空")
		return
	}

	// 检查用户名是否已存在
	existing, err := h.userRepo.GetByUsername(r.Context(), req.Username)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询用户失败")
		return
	}
	if existing != nil {
		handler.WriteError(w, http.StatusConflict, "用户名已存在")
		return
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "密码加密失败")
		return
	}

	// 设置默认角色
	role := req.Role
	if role == 0 {
		role = middleware.RoleViewer
	}

	// 创建用户
	user := &database.User{
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		DisplayName:  req.DisplayName,
		Email:        req.Email,
		Role:         role,
		Status:       1, // 默认启用
	}

	if err := h.userRepo.Create(r.Context(), user); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "创建用户失败: "+err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "注册成功",
		"data": UserInfo{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Email:       user.Email,
			Role:        user.Role,
		},
	})
}

// List 获取用户列表
// GET /api/v2/user/list
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	users, err := h.userRepo.List(r.Context())
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "获取用户列表失败")
		return
	}

	// 转换为 UserDTO
	result := make([]UserDTO, 0, len(users))
	for _, u := range users {
		dto := UserDTO{
			ID:          u.ID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			Email:       u.Email,
			Role:        u.Role,
			Status:      u.Status,
			CreatedAt:   u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		result = append(result, dto)
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    result,
	})
}

// UpdateRole 更新用户角色
// POST /api/v2/user/update-role
func (h *UserHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "请求参数无效")
		return
	}

	if req.UserID == 0 {
		handler.WriteError(w, http.StatusBadRequest, "用户 ID 不能为空")
		return
	}

	if req.Role < 1 || req.Role > 3 {
		handler.WriteError(w, http.StatusBadRequest, "无效的角色值")
		return
	}

	// 查找用户
	user, err := h.userRepo.GetByID(r.Context(), req.UserID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询用户失败")
		return
	}
	if user == nil {
		handler.WriteError(w, http.StatusNotFound, "用户不存在")
		return
	}

	// 更新角色
	user.Role = req.Role
	if err := h.userRepo.Update(r.Context(), user); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "更新角色失败")
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "角色更新成功",
	})
}

// Delete 删除用户
// POST /api/v2/user/delete
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req DeleteUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "请求参数无效")
		return
	}

	if req.UserID == 0 {
		handler.WriteError(w, http.StatusBadRequest, "用户 ID 不能为空")
		return
	}

	// 获取当前操作者 ID
	operatorID, ok := middleware.GetUserID(r.Context())
	if !ok {
		handler.WriteError(w, http.StatusUnauthorized, "无法获取当前用户信息")
		return
	}

	// 不能删除自己
	if req.UserID == operatorID {
		handler.WriteError(w, http.StatusBadRequest, "不能删除自己")
		return
	}

	// 检查用户是否存在
	user, err := h.userRepo.GetByID(r.Context(), req.UserID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询用户失败")
		return
	}
	if user == nil {
		handler.WriteError(w, http.StatusNotFound, "用户不存在")
		return
	}

	// 不能删除 admin 用户
	if user.Username == "admin" {
		handler.WriteError(w, http.StatusForbidden, "admin 用户不可删除")
		return
	}

	// 删除用户
	if err := h.userRepo.Delete(r.Context(), req.UserID); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "删除用户失败")
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "用户删除成功",
	})
}

// UpdateStatus 更新用户状态（启用/禁用）
// POST /api/v2/user/update-status
func (h *UserHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "请求参数无效")
		return
	}

	if req.UserID == 0 {
		handler.WriteError(w, http.StatusBadRequest, "用户 ID 不能为空")
		return
	}

	if req.Status != 0 && req.Status != 1 {
		handler.WriteError(w, http.StatusBadRequest, "无效的状态值，必须为 0 或 1")
		return
	}

	// 获取当前操作者 ID
	operatorID, ok := middleware.GetUserID(r.Context())
	if !ok {
		handler.WriteError(w, http.StatusUnauthorized, "无法获取当前用户信息")
		return
	}

	// 不能禁用自己
	if req.UserID == operatorID && req.Status == 0 {
		handler.WriteError(w, http.StatusBadRequest, "不能禁用自己")
		return
	}

	// 查找用户
	user, err := h.userRepo.GetByID(r.Context(), req.UserID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询用户失败")
		return
	}
	if user == nil {
		handler.WriteError(w, http.StatusNotFound, "用户不存在")
		return
	}

	// 不能禁用 admin 用户
	if user.Username == "admin" && req.Status == 0 {
		handler.WriteError(w, http.StatusForbidden, "admin 用户不可禁用")
		return
	}

	// 更新状态
	user.Status = req.Status
	if err := h.userRepo.Update(r.Context(), user); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "更新状态失败")
		return
	}

	statusText := "已启用"
	if req.Status == 0 {
		statusText = "已禁用"
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "用户" + statusText,
	})
}
