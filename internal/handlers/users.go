package handlers

import (
	"net/http"
	"strconv"
	"time"

	"bank-api/internal/models"
	"bank-api/internal/validation"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email,max=100"`
	Password string `json:"password" binding:"required,min=8,max=100"`
	FullName string `json:"fullName" binding:"required,min=2,max=100"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) RegisterUser(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate registration input
	validator := validation.New()
	if err := validator.ValidateUserRegistration(req.Username, req.Email, req.Password, req.FullName); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	var existingUser models.User
	if err := h.DB.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "username or email already exists"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	user.PasswordHash = string(hashedPassword)

	if err := h.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// Log audit event
	if err := h.AuditService.LogUserRegistration(c, &user); err != nil {
		c.Error(err)
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"fullName": user.FullName,
	})
}

func (h *Handler) LoginUser(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(h.Config.JWT.Expiration).Unix(),
	})

	tokenString, err := token.SignedString([]byte(h.Config.JWT.Secret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"fullName": user.FullName,
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
	userID := c.GetUint("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"username": user.Username,
		"name":     user.FullName,
		"email":    user.Email,
	})
}

func (h *Handler) handleTestUser(c *gin.Context) {
	var user models.User
	idParam := c.Param("id")
	userID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	if err := h.DB.First(&user, uint(userID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"username":  user.Username,
		"full_name": user.FullName,
		"email":     user.Email,
	})
}

func (h *Handler) UpdateUser(c *gin.Context) {
	userID := c.GetUint("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		}
		return
	}

	oldUser := user

	var updateData struct {
		FullName string `json:"fullName"`
		Email    string `json:"email"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	validator := validation.New()
	if updateData.FullName != "" {
		if err := validator.ValidateFullName(updateData.FullName); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		user.FullName = updateData.FullName
	}
	if updateData.Email != "" {
		if err := validator.ValidateEmail(updateData.Email); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email format"})
			return
		}

		var existingUser models.User
		if err := h.DB.Where("email = ? AND id != ?", updateData.Email, userID).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "email already in use"})
			return
		}
		user.Email = updateData.Email
	}

	tx := h.DB.Begin()
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}
	tx.Commit()

	// Log audit event for user update
	if err := h.AuditService.LogUserAction(c, userID, "update", &oldUser, &user); err != nil {
		c.Error(err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User updated successfully",
	})
}

func (h *Handler) DeleteUser(c *gin.Context) {
	userID := c.GetUint("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user accounts"})
		return
	}

	// Soft-delete the user
	if err := tx.Delete(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}

	// Invalidate account count cache
	h.AccountCountCache.Delete(userID)

	// Log audit event for user deletion
	if err := h.AuditService.LogUserAction(c, userID, "delete", &user, nil); err != nil {
		c.Error(err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User deleted successfully",
	})
}

// validatePassword validates password complexity requirements
