package handlers

import (
	"net/http"
	"time"

	"bank-api/internal/models"
	"bank-api/internal/validation"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	FullName string `json:"fullName"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type UserProfileResponse struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

type TestUserResponse struct {
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email,max=100"`
	Password string `json:"password" binding:"required,min=8,max=100"`
	FullName string `json:"fullName" binding:"required,min=2,max=100"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=8,max=100"`
}

func (h *Handler) RegisterUser(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	// Validate registration input
	validator := validation.New()
	if err := validator.ValidateUserRegistration(req.Username, req.Email, req.Password, req.FullName); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
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

	// Log audit event
	if err := h.AuditService.LogUserRegistration(c, &user); err != nil {
		c.Error(err)
	}

	c.JSON(http.StatusCreated, UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		FullName: user.FullName,
	})
}

func (h *Handler) LoginUser(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	// Validate username and password against control characters, line breaks, and null bytes to prevent injection/truncation
	validator := validation.New()
	if err := validator.ValidateUsername(req.Username); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	for i := 0; i < len(req.Password); i++ {
		ch := req.Password[i]
		if ch == 0 || ch == '\r' || ch == '\n' {
			c.JSON(http.StatusBadRequest, errorResponse{Error: "password cannot contain null bytes or line breaks"})
			return
		}
	}

	var user models.User
	err := h.DB.Where("username = ?", req.Username).First(&user).Error

	// If user not found, use dummy hash for comparison to prevent timing attacks
	targetHash := user.PasswordHash
	if err != nil {
		targetHash = h.dummyHash
	}

	// Always perform bcrypt comparison to maintain constant time (roughly)
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

	c.JSON(http.StatusOK, LoginResponse{
		Token: tokenString,
		User: UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			FullName: user.FullName,
		},
	})
}

func (h *Handler) GetUser(c *gin.Context) {
	// Handle legacy test mode route /users/:id if id param is present
	if gin.Mode() == gin.TestMode && c.Param("id") != "" {
		h.handleTestUser(c)
		return
	}

	// Production mode logic
	userID := getUserID(c)
	if userID == 0 {
		c.Data(http.StatusUnauthorized, "application/json", unauthorizedErrorJSON)
		return
	}

	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.Data(http.StatusNotFound, "application/json; charset=utf-8", userNotFoundErrorJSON)
		} else {
			c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", databaseErrorJSON)
		}
		return
	}

	c.JSON(http.StatusOK, UserProfileResponse{
		Username: user.Username,
		Name:     user.FullName,
		Email:    user.Email,
	})
}

func (h *Handler) handleTestUser(c *gin.Context) {
	var user models.User
	idParam := c.Param("id")
	userID, err := validation.ParseUint(idParam, 0)
	if err != nil {
		c.Data(http.StatusBadRequest, "application/json; charset=utf-8", invalidUserIDErrorJSON)
		return
	}

	if err := h.DB.First(&user, uint(userID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.Data(http.StatusNotFound, "application/json; charset=utf-8", userNotFoundErrorJSON)
		} else {
			c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", databaseErrorJSON)
		}
		return
	}

	c.JSON(http.StatusOK, TestUserResponse{
		Username: user.Username,
		FullName: user.FullName,
		Email:    user.Email,
	})
}

func (h *Handler) UpdateUser(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		c.Data(http.StatusUnauthorized, "application/json", unauthorizedErrorJSON)
		return
	}

	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.Data(http.StatusNotFound, "application/json; charset=utf-8", userNotFoundErrorJSON)
		} else {
			c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", databaseErrorJSON)
		}
		return
	}

	oldUser := user

	var updateData struct {
		FullName *string `json:"fullName" binding:"omitempty,min=2,max=100"`
		Email    *string `json:"email" binding:"omitempty,email,max=100"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.Data(http.StatusBadRequest, "application/json; charset=utf-8", invalidRequestErrorJSON)
		return
	}

	if updateData.FullName == nil && updateData.Email == nil {
		c.Data(http.StatusBadRequest, "application/json; charset=utf-8", atLeastOneFieldMustBeProvidedErrorJSON)
		return
	}

	validator := validation.New()
	if updateData.FullName != nil {
		if err := validator.ValidateFullName(*updateData.FullName); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		user.FullName = *updateData.FullName
	}
	if updateData.Email != nil {
		if err := validator.ValidateEmail(*updateData.Email); err != nil {
			c.Data(http.StatusBadRequest, "application/json; charset=utf-8", invalidEmailFormatErrorJSON)
			return
		}

		var existingUser models.User
		if err := h.DB.Where("email = ? AND id != ?", *updateData.Email, userID).First(&existingUser).Error; err == nil {
			c.Data(http.StatusConflict, "application/json; charset=utf-8", emailAlreadyInUseErrorJSON)
			return
		}
		user.Email = *updateData.Email
	}

	tx := h.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToUpdateUserErrorJSON)
		return
	}
	if err := tx.Commit().Error; err != nil {
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToUpdateUserErrorJSON)
		return
	}

	// Log audit event for user update
	if err := h.AuditService.LogUserAction(c, userID, "update", &oldUser, &user); err != nil {
		c.Error(err)
	}

	c.Data(http.StatusOK, "application/json; charset=utf-8", userUpdatedSuccessfullyJSON)
}

func (h *Handler) DeleteUser(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		c.Data(http.StatusUnauthorized, "application/json", unauthorizedErrorJSON)
		return
	}

	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.Data(http.StatusNotFound, "application/json; charset=utf-8", userNotFoundErrorJSON)
		} else {
			c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", databaseErrorJSON)
		}
		return
	}

	// Begin database transaction for atomic deletion
	tx := h.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Soft-delete all bank accounts associated with the user to prevent orphaned accounts
	if err := tx.Where("user_id = ?", userID).Delete(&models.Account{}).Error; err != nil {
		tx.Rollback()
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToDeleteUserAccountsErrorJSON)
		return
	}

	// Soft-delete the user
	if err := tx.Delete(&user).Error; err != nil {
		tx.Rollback()
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToDeleteUserErrorJSON)
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToDeleteUserErrorJSON)
		return
	}

	// Invalidate account count cache
	h.AccountCountCache.Delete(userID)

	// Log audit event for user deletion
	if err := h.AuditService.LogUserAction(c, userID, "delete", &user, nil); err != nil {
		c.Error(err)
	}

	c.Data(http.StatusOK, "application/json; charset=utf-8", userDeletedSuccessfullyJSON)
}

// validatePassword validates password complexity requirements
