package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	httpreq "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/request"
	httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

// AuthHandler serves user registration, login, and API key management endpoints.
type AuthHandler struct {
	svc        *service.AuthService
	configPath string
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(svc *service.AuthService, configPath string) *AuthHandler {
	return &AuthHandler{svc: svc, configPath: configPath}
}

// RegisterRoutes mounts public auth endpoints (no JWT required).
func (h *AuthHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.POST("/auth/login", h.Login)
}

// RegisterProtectedRoutes mounts authenticated auth endpoints (JWT required).
func (h *AuthHandler) RegisterProtectedRoutes(api *gin.RouterGroup) {
	api.GET("/auth/me", h.Me)
	api.PUT("/auth/password", h.ChangePassword)
	api.GET("/auth/api-keys", h.ListAPIKeys)
	api.POST("/auth/api-keys", h.AddAPIKey)
	api.DELETE("/auth/api-keys/:key", h.DeleteAPIKey)
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	token, user, err := h.svc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if err == service.ErrInvalidCreds {
			httpresp.Error(c, http.StatusUnauthorized, err.Error())
			return
		}
		httpresp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	httpresp.OK(c, gin.H{
		"token": token,
		"user":  user,
	})
}

// Me handles GET /auth/me — returns the authenticated user's profile.
func (h *AuthHandler) Me(c *gin.Context) {
	actor := httpreq.From(c)
	userID, err := strconv.ParseUint(actor.UserID, 10, 64)
	if err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "invalid user identity")
		return
	}

	user, err := h.svc.GetUserByID(c.Request.Context(), uint(userID))
	if err != nil {
		httpresp.Error(c, http.StatusNotFound, err.Error())
		return
	}

	httpresp.OK(c, user)
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ChangePassword handles PUT /auth/password — updates the authenticated user's password.
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	actor := httpreq.From(c)
	userID, err := strconv.ParseUint(actor.UserID, 10, 64)
	if err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "invalid user identity")
		return
	}

	if err := h.svc.ChangePassword(c.Request.Context(), uint(userID), req.OldPassword, req.NewPassword); err != nil {
		switch {
		case err == service.ErrInvalidCreds:
			httpresp.Error(c, http.StatusUnauthorized, "旧密码不正确")
			return
		case err == service.ErrSamePassword:
			httpresp.Error(c, http.StatusBadRequest, "新密码不能与旧密码相同")
			return
		case err == service.ErrPasswordTooShort:
			httpresp.Error(c, http.StatusBadRequest, "密码长度不能少于6位")
			return
		default:
			httpresp.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	httpresp.OK(c, gin.H{"message": "密码修改成功"})
}

// ListAPIKeys handles GET /auth/api-keys.
func (h *AuthHandler) ListAPIKeys(c *gin.Context) {
	cfg := config.Get()
	httpresp.OK(c, cfg.Auth.APIKeys)
}

type addAPIKeyRequest struct {
	Key string `json:"key" binding:"required"`
}

// AddAPIKey handles POST /auth/api-keys.
func (h *AuthHandler) AddAPIKey(c *gin.Context) {
	var req addAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := config.Save(h.configPath, func(cfg *config.Config) {
		for _, k := range cfg.Auth.APIKeys {
			if k == req.Key {
				return
			}
		}
		cfg.Auth.APIKeys = append(cfg.Auth.APIKeys, req.Key)
	}); err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "save config failed: "+err.Error())
		return
	}

	httpresp.Created(c, gin.H{"message": "API key added"})
}

// DeleteAPIKey handles DELETE /auth/api-keys/:key.
func (h *AuthHandler) DeleteAPIKey(c *gin.Context) {
	key := strings.TrimSpace(c.Param("key"))
	if key == "" {
		httpresp.Error(c, http.StatusBadRequest, "key is required")
		return
	}

	if err := config.Save(h.configPath, func(cfg *config.Config) {
		for i, k := range cfg.Auth.APIKeys {
			if k == key {
				cfg.Auth.APIKeys = append(cfg.Auth.APIKeys[:i], cfg.Auth.APIKeys[i+1:]...)
				return
			}
		}
	}); err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "save config failed: "+err.Error())
		return
	}

	httpresp.OK(c, gin.H{"message": "API key deleted"})
}
