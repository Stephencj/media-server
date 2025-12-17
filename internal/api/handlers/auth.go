package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stephencjuliano/media-server/internal/config"
	"github.com/stephencjuliano/media-server/internal/db"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	db  *db.DB
	cfg *config.Config
}

func NewAuthHandler(database *db.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{db: database, cfg: cfg}
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type TokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	User      struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	} `json:"user"`
}

// Register creates a new user account
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if username already exists
	if _, err := h.db.GetUserByUsername(req.Username); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}

	// Check if email already exists
	if _, err := h.db.GetUserByEmail(req.Email); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Create user
	user, err := h.db.CreateUser(req.Username, req.Email, string(hashedPassword))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Generate token
	response, err := h.generateTokenResponse(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// Login authenticates a user and returns a JWT token
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user
	user, err := h.db.GetUserByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate token
	response, err := h.generateTokenResponse(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// RefreshToken generates a new token from an existing valid token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, err := h.db.GetUserByID(userID.(int64))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	response, err := h.generateTokenResponse(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *AuthHandler) generateTokenResponse(user *db.User) (*TokenResponse, error) {
	expiresAt := time.Now().Add(time.Duration(h.cfg.JWTExpiration) * time.Hour)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
		"exp":      expiresAt.Unix(),
		"iat":      time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	response := &TokenResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt.Unix(),
	}
	response.User.ID = user.ID
	response.User.Username = user.Username
	response.User.Email = user.Email

	return response, nil
}
