package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stephencjuliano/media-server/internal/config"
	"github.com/stephencjuliano/media-server/internal/db"
	"golang.org/x/crypto/bcrypt"
)

const webLoginCodeTTL = 60 * time.Second

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

// ExchangeKeyRequest is the body for /api/auth/exchange.
type ExchangeKeyRequest struct {
	Key string `json:"key" binding:"required"`
}

// ExchangeDeviceKey validates a per-platform device API key and returns a JWT.
// This is the auto-auth path used by iOS / tvOS / Fire TV builds — no
// username or password ever leaves the client.
func (h *AuthHandler) ExchangeDeviceKey(c *gin.Context) {
	var req ExchangeKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sum := sha256.Sum256([]byte(req.Key))
	keyHash := hex.EncodeToString(sum[:])

	dk, err := h.db.GetDeviceKeyByHash(keyHash)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid device key"})
		return
	}

	user, err := h.db.GetUserByID(dk.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Device owner not found"})
		return
	}

	_ = h.db.TouchDeviceKey(dk.ID)

	response, err := h.generateTokenResponse(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// WebLoginCreateResponse is the body of /api/auth/web-login/create.
type WebLoginCreateResponse struct {
	Code      string `json:"code"`
	ExpiresAt int64  `json:"expires_at"`
}

// CreateWebLoginCode mints a fresh single-use code that the web admin can
// embed in a QR. Public — no auth required, since the code is useless until
// an authenticated iOS device claims it.
func (h *AuthHandler) CreateWebLoginCode(c *gin.Context) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate code"})
		return
	}
	code := hex.EncodeToString(raw[:])

	sum := sha256.Sum256([]byte(code))
	hash := hex.EncodeToString(sum[:])
	expires := time.Now().Add(webLoginCodeTTL)

	if err := h.db.CreateWebLoginCode(hash, expires); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist code"})
		return
	}

	c.JSON(http.StatusOK, WebLoginCreateResponse{
		Code:      code,
		ExpiresAt: expires.Unix(),
	})
}

// WebLoginClaimRequest is the body for /api/auth/web-login/claim.
type WebLoginClaimRequest struct {
	Code string `json:"code" binding:"required"`
}

// ClaimWebLoginCode is called by an authenticated iOS device after scanning
// a QR. It binds the code to the JWT user so the next browser poll succeeds.
func (h *AuthHandler) ClaimWebLoginCode(c *gin.Context) {
	var req WebLoginClaimRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	userID, _ := userIDVal.(int64)

	sum := sha256.Sum256([]byte(req.Code))
	hash := hex.EncodeToString(sum[:])

	if err := h.db.ClaimWebLoginCode(hash, userID); err != nil {
		c.JSON(http.StatusGone, gin.H{"error": "Code unknown, expired, or already claimed"})
		return
	}

	c.Status(http.StatusNoContent)
}

// PollWebLoginCode is called by the browser every ~2s while the QR is shown.
// Returns the standard TokenResponse on the first poll after claim, then 410
// for any subsequent polls (single-use).
func (h *AuthHandler) PollWebLoginCode(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code query parameter required"})
		return
	}

	sum := sha256.Sum256([]byte(code))
	hash := hex.EncodeToString(sum[:])

	result, user, err := h.db.ConsumeWebLoginCode(hash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read code"})
		return
	}

	switch result {
	case db.WebLoginNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": "Unknown code"})
	case db.WebLoginPending:
		c.Status(http.StatusNoContent)
	case db.WebLoginExpired:
		c.JSON(http.StatusGone, gin.H{"error": "Code expired or already used"})
	case db.WebLoginReady:
		response, err := h.generateTokenResponse(user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}
		c.JSON(http.StatusOK, response)
	}
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
