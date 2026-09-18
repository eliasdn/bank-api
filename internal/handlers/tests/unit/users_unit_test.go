package unit

import (
	"bank-api/internal/audit"
	"bank-api/internal/config"
	"bank-api/internal/db"
	"bank-api/internal/handlers"
	"bank-api/internal/models"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type UsersUnitTestSuite struct {
	suite.Suite
	handler *handlers.Handler
	db      *gorm.DB
	router  *gin.Engine
}

func (suite *UsersUnitTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	var err error
	suite.db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	// Migrate the schema
	suite.db.AutoMigrate(&models.User{}, &models.Account{}, &audit.AuditLog{})

	// Create a handler with the real database
	testConfig := config.LoadTestConfig()
	suite.handler = handlers.NewHandler(&db.Database{DB: suite.db}, testConfig)
	suite.router = gin.New()

	// Register test routes without auth middleware
	suite.router.GET("/users/:id", suite.handler.GetUser)
	suite.router.POST("/register", suite.handler.RegisterUser)
	suite.router.POST("/login", suite.handler.LoginUser)

	// Auth mock route group for user update and delete testing
	authed := suite.router.Group("/users/me")
	authed.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	authed.PUT("", suite.handler.UpdateUser)
	authed.DELETE("", suite.handler.DeleteUser)
}

func (suite *UsersUnitTestSuite) TearDownTest() {
	time.Sleep(50 * time.Millisecond)
}

func (suite *UsersUnitTestSuite) TestRegisterUser_ValidationFailure() {
	// Test registration with invalid username characters (e.g., spaces or special symbols)
	invalidReq := map[string]string{
		"username": "invalid user!",
		"email":    "user@example.com",
		"password": "Password123!",
		"fullName": "Test User",
	}
	body, _ := json.Marshal(invalidReq)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *UsersUnitTestSuite) TestGetUser_Success() {
	// Create a test user
	testUser := &models.User{
		Model:    gorm.Model{ID: 1},
		Username: "testuser",
		FullName: "Test User",
	}
	suite.db.Create(testUser)

	// Make request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/users/1", nil)
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), response["username"])
	assert.NotEmpty(suite.T(), response["full_name"])
}

func (suite *UsersUnitTestSuite) TestGetUser_NotFound() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/users/999", nil) // Use non-existent ID
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

func (suite *UsersUnitTestSuite) TestUpdateUser_SuccessAndAuditLog() {
	testUser := &models.User{
		Model:    gorm.Model{ID: 1},
		Username: "testuser",
		FullName: "Old Name",
		Email:    "old@example.com",
	}
	suite.db.Create(testUser)

	updateReq := map[string]string{
		"fullName": "Updated Name",
		"email":    "updated@example.com",
	}
	body, _ := json.Marshal(updateReq)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/users/me", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var updatedUser models.User
	suite.db.First(&updatedUser, 1)
	assert.Equal(suite.T(), "Updated Name", updatedUser.FullName)
	assert.Equal(suite.T(), "updated@example.com", updatedUser.Email)

	var auditLog audit.AuditLog
	err := suite.db.Where("user_id = ? AND action = ?", 1, "update").First(&auditLog).Error
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "user", auditLog.Resource)
}

func (suite *UsersUnitTestSuite) TestUserJSON_ExcludesPasswordHash() {
	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "$2a$10$SecretPasswordHashValueNotToExpose",
		FullName:     "Test User",
	}

	data, err := json.Marshal(user)
	assert.NoError(suite.T(), err)
	assert.NotContains(suite.T(), string(data), "PasswordHash")
	assert.NotContains(suite.T(), string(data), "SecretPasswordHashValueNotToExpose")
}

func (suite *UsersUnitTestSuite) TestLoginUser_ContainsIssuedAtClaim() {
	password := "Password123!"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), 4)
	testUser := &models.User{
		Model:        gorm.Model{ID: 10},
		Username:     "jwtuser",
		Email:        "jwt@example.com",
		PasswordHash: string(hashedPassword),
		FullName:     "JWT User",
	}
	suite.db.Create(testUser)

	loginReq := map[string]string{
		"username": "jwtuser",
		"password": password,
	}
	body, _ := json.Marshal(loginReq)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var res map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &res)
	assert.NoError(suite.T(), err)

	tokenString, ok := res["token"].(string)
	assert.True(suite.T(), ok)

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return []byte(config.LoadTestConfig().JWT.Secret), nil
	})
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), token.Valid)

	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(suite.T(), ok)

	iat, exists := claims["iat"]
	assert.True(suite.T(), exists)
	assert.NotNil(suite.T(), iat)
}

func (suite *UsersUnitTestSuite) TestDeleteUser_SuccessAndAuditLog() {
	testUser := &models.User{
		Model:    gorm.Model{ID: 1},
		Username: "user2delete",
		FullName: "User To Delete",
		Email:    "delete@example.com",
	}
	suite.db.Create(testUser)

	testAccount := &models.Account{
		Model:         gorm.Model{ID: 100},
		UserID:        1,
		AccountNumber: "ACC999888",
		AccountType:   "checking",
		Balance:       500,
	}
	suite.db.Create(testAccount)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/users/me", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var deletedUser models.User
	err := suite.db.First(&deletedUser, 1).Error
	assert.ErrorIs(suite.T(), err, gorm.ErrRecordNotFound)

	var deletedAccount models.Account
	err = suite.db.First(&deletedAccount, 100).Error
	assert.ErrorIs(suite.T(), err, gorm.ErrRecordNotFound)

	var auditLog audit.AuditLog
	err = suite.db.Where("user_id = ? AND action = ?", 1, "delete").First(&auditLog).Error
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "user", auditLog.Resource)
}

func TestUsersUnitSuite(t *testing.T) {
	suite.Run(t, new(UsersUnitTestSuite))
}
