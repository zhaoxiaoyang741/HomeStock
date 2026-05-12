package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	httpreq "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/request"
	httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
)

// AuthHandler serves user registration and login endpoints.
type AuthHandler struct {
	svc *service.AuthService
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// RegisterRoutes mounts public auth endpoints (no JWT required).
func (h *AuthHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.POST("/auth/login", h.Login)
}

// RegisterProtectedRoutes mounts authenticated auth endpoints (JWT required).
func (h *AuthHandler) RegisterProtectedRoutes(api *gin.RouterGroup) {
	api.GET("/auth/me", h.Me)
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
