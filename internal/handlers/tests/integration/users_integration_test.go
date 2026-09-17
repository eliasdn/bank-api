package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bank-api/internal/config"
	"bank-api/internal/db"
	"bank-api/internal/handlers"
	"bank-api/internal/middleware"
	"bank-api/internal/models"
	"bank-api/internal/testutils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type UsersIntegrationTestSuite struct {
	suite.Suite
	handler *handlers.Handler
	db      db.DBInterface
	router  *gin.Engine
	cfg     *config.AppConfig
}

func (suite *UsersIntegrationTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)

	gormDB := testutils.InitTestDB()
	suite.db = db.NewDBAdapter(gormDB)

	suite.cfg = config.LoadTestConfig()
	suite.cfg.JWT.Secret = "test-jwt-secret"
	suite.handler = handlers.NewHandler(suite.db, suite.cfg)

	suite.router = gin.New()
	authMiddleware := middleware.NewAuthMiddleware(suite.cfg)

	api := suite.router.Group("/api/v1")
	users := api.Group("/users")
	users.Use(authMiddleware.Authenticate())
	{
		users.GET("/me", suite.handler.GetUser)
		users.PUT("/me", suite.handler.UpdateUser)
		users.DELETE("/me", suite.handler.DeleteUser)
	}
}

func (suite *UsersIntegrationTestSuite) generateToken(userID uint) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(suite.cfg.JWT.Secret))
	assert.NoError(suite.T(), err)
	return tokenStr
}

func (suite *UsersIntegrationTestSuite) TestGetUser_Success() {
	user := &models.User{
		Model:    gorm.Model{ID: 10},
		Username: "john_doe",
		FullName: "John Doe",
		Email:    "john@example.com",
	}
	suite.db.Create(user)

	token := suite.generateToken(10)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var res map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &res)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "john_doe", res["username"])
	assert.Equal(suite.T(), "John Doe", res["name"])
	assert.Equal(suite.T(), "john@example.com", res["email"])
}

func (suite *UsersIntegrationTestSuite) TestGetUser_Unauthorized() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/users/me", nil)

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *UsersIntegrationTestSuite) TestUpdateUser_Success() {
	user := &models.User{
		Model:    gorm.Model{ID: 20},
		Username: "jane_doe",
		FullName: "Jane Doe",
		Email:    "jane@example.com",
	}
	suite.db.Create(user)

	token := suite.generateToken(20)

	body := map[string]string{
		"fullName": "Jane Updated",
		"email":    "jane_updated@example.com",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/users/me", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var updatedUser models.User
	err := suite.db.First(&updatedUser, 20).Error
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Jane Updated", updatedUser.FullName)
	assert.Equal(suite.T(), "jane_updated@example.com", updatedUser.Email)
}

func (suite *UsersIntegrationTestSuite) TestUpdateUser_FullNameValidationFailure() {
	user := &models.User{
		Model:    gorm.Model{ID: 25},
		Username: "validation_user",
		FullName: "Valid Name",
		Email:    "val@example.com",
	}
	suite.db.Create(user)

	token := suite.generateToken(25)

	body := map[string]string{
		"fullName": "a", // Too short (min length is 2)
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/users/me", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *UsersIntegrationTestSuite) TestDeleteUser_Success() {
	user := &models.User{
		Model:    gorm.Model{ID: 30},
		Username: "to_delete",
		FullName: "To Delete",
		Email:    "delete@example.com",
	}
	suite.db.Create(user)

	token := suite.generateToken(30)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var count int64
	suite.db.Model(&models.User{}).Where("id = ?", 30).Count(&count)
	assert.Equal(suite.T(), int64(0), count)
}

func TestUsersIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(UsersIntegrationTestSuite))
}
