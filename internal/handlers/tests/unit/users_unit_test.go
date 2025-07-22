package unit

import (
	"bank-api/internal/config"
	"bank-api/internal/db"
	"bank-api/internal/handlers"
	"bank-api/internal/models"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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
	suite.db.AutoMigrate(&models.User{})

	// Create a handler with the real database
	testConfig := config.LoadTestConfig()
	suite.handler = handlers.NewHandler(&db.Database{DB: suite.db}, testConfig)
	suite.router = gin.New()

	// Register test routes without auth middleware
	suite.router.GET("/users/:id", suite.handler.GetUser)
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

func TestUsersUnitSuite(t *testing.T) {
	suite.Run(t, new(UsersUnitTestSuite))
}
