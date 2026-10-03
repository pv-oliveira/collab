package handlers

import (
	"apis/internal/middleware"
	"apis/internal/models"
	"apis/internal/services"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	Service *services.AuthService
	Tokens  *middleware.Tokens
}

func (h *AuthHandler) Register(c *gin.Context) {
	var body struct {
		Email    string `form:"email" binding:"required"`
		Password string `form:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.Service.Register(body.Email, body.Password)
	if errors.Is(err, models.ErrEmailTaken) {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}
	if err != nil {
		log.Println("register:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.Service.Login(body.Email, body.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// Logout revoga o token usado nesta requisição (rota protegida pelo
// AuthMiddleware, que deixa as claims no contexto).
func (h *AuthHandler) Logout(c *gin.Context) {
	claims := c.MustGet("claims").(*middleware.Claims)
	if err := h.Tokens.Revoke(c, claims); err != nil {
		log.Println("logout:", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "service unavailable"})
		return
	}
	c.Status(http.StatusNoContent)
}
