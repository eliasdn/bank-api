package auth

import (
	"bank-api/internal/config"
	"bank-api/internal/db"
	"bank-api/internal/models"
	"bank-api/internal/validation"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	usernameOrEmailAlreadyExistsErrorJSON = []byte(`{"error":"username or email already exists"}`)
	failedToHashPasswordErrorJSON         = []byte(`{"error":"failed to hash password"}`)
	failedToCreateUserErrorJSON           = []byte(`{"error":"failed to create user"}`)
	invalidCredentialsErrorJSON           = []byte(`{"error":"invalid credentials"}`)
	failedToGenerateTokenErrorJSON        = []byte(`{"error":"failed to generate token"}`)
	invalidPasswordFormatErrorJSON        = []byte(`{"error":"password cannot contain null bytes or line breaks"}`)
)

type Handler struct {
	DB        db.DBInterface
	Config    *config.AppConfig
	dummyHash string
	dummyOnce sync.Once
}

func (h *Handler) getDummyHash() string {
	h.dummyOnce.Do(func() {
		if h.dummyHash == "" {
			cost := bcrypt.DefaultCost
			if h.Config != nil && h.Config.Security.BcryptCost > 0 {
				cost = h.Config.Security.BcryptCost
			}
			hash, _ := bcrypt.GenerateFromPassword([]byte("dummy_password_for_timing_prevention"), cost)
			h.dummyHash = string(hash)
		}
	})
	return h.dummyHash
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email,max=100"`
	Password string `json:"password" binding:"required,min=8,max=100"`
	FullName string `json:"fullName" binding:"required,min=2,max=100"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required,max=50"`
	Password string `json:"password" binding:"required,max=100"`
}

func (h *Handler) RegisterUser(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, struct {
			Error string `json:"error"`
		}{Error: err.Error()})
		return
	}

	// Validate registration input
	validator := validation.New()
	if err := validator.ValidateUserRegistration(req.Username, req.Email, req.Password, req.FullName); err != nil {
		c.JSON(http.StatusBadRequest, struct {
			Error string `json:"error"`
		}{Error: err.Error()})
		return
	}

	// Check if user already exists
	var existingUser models.User
	if err := h.DB.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error; err == nil {
		c.Data(http.StatusConflict, "application/json; charset=utf-8", usernameOrEmailAlreadyExistsErrorJSON)
		return
	}

	// Create new user
	user := models.User{
		Username: req.Username,
		Email:    req.Email,
		FullName: req.FullName,
	}

	// Hash password with configurable cost
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), h.Config.Security.BcryptCost)
	if err != nil {
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToHashPasswordErrorJSON)
		return
	}
	user.PasswordHash = string(hashedPassword)

	if err := h.DB.Create(&user).Error; err != nil {
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToCreateUserErrorJSON)
		return
	}

	c.JSON(http.StatusCreated, struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		FullName string `json:"fullName"`
	}{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		FullName: user.FullName,
	})
}

func (h *Handler) LoginUser(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, struct {
			Error string `json:"error"`
		}{Error: err.Error()})
		return
	}

	// Validate username and password against control characters, line breaks, and null bytes to prevent injection/truncation
	validator := validation.New()
	if err := validator.ValidateUsername(req.Username); err != nil {
		c.JSON(http.StatusBadRequest, struct {
			Error string `json:"error"`
		}{Error: err.Error()})
		return
	}
	for i := 0; i < len(req.Password); i++ {
		ch := req.Password[i]
		if ch == 0 || ch == '\r' || ch == '\n' {
			c.Data(http.StatusBadRequest, "application/json; charset=utf-8", invalidPasswordFormatErrorJSON)
			return
		}
	}

	var user models.User
	err := h.DB.Where("username = ?", req.Username).First(&user).Error

	// If user not found, use dummy hash for comparison to prevent timing attacks
	targetHash := user.PasswordHash
	if err != nil {
		targetHash = h.getDummyHash()
	}

	// Always perform bcrypt comparison to maintain constant time
	bcryptErr := bcrypt.CompareHashAndPassword([]byte(targetHash), []byte(req.Password))

	if err != nil || bcryptErr != nil {
		c.Data(http.StatusUnauthorized, "application/json; charset=utf-8", invalidCredentialsErrorJSON)
		return
	}

	// Generate JWT token
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"iat": now.Unix(),
		"exp": now.Add(h.Config.JWT.Expiration).Unix(),
	})

	tokenString, err := token.SignedString([]byte(h.Config.JWT.Secret))
	if err != nil {
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToGenerateTokenErrorJSON)
		return
	}

	c.JSON(http.StatusOK, struct {
		Token string `json:"token"`
		User  struct {
			ID       uint   `json:"id"`
			Username string `json:"username"`
			Email    string `json:"email"`
			FullName string `json:"fullName"`
		} `json:"user"`
	}{
		Token: tokenString,
		User: struct {
			ID       uint   `json:"id"`
			Username string `json:"username"`
			Email    string `json:"email"`
			FullName string `json:"fullName"`
		}{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			FullName: user.FullName,
		},
	})
}
