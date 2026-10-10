package services

import (
	"bank-api/internal/audit"
	"bank-api/internal/models"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLogTransaction_NoContextRace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&audit.AuditLog{})
	assert.NoError(t, err)

	service := NewAuditService(db)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)
	c.Request.Header.Set("User-Agent", "Test-Agent")

	transaction := &models.Transaction{
		Model:           gorm.Model{ID: 101},
		AccountID:       1,
		Amount:          250.0,
		TransactionType: "deposit",
		Reference:       "Test deposit",
		Status:          "completed",
	}
	account := &models.Account{
		Model:         gorm.Model{ID: 1},
		UserID:        1,
		AccountNumber: "ACC999",
		Balance:       1250.0,
	}

	err = service.LogTransaction(c, 1, transaction, account)
	assert.NoError(t, err)

	// Wait briefly for background goroutine to complete
	time.Sleep(50 * time.Millisecond)

	// Verify gin.Context was not mutated with errors
	assert.Empty(t, c.Errors)

	// Verify audit log entry was created
	var log audit.AuditLog
	err = db.Where("resource_id = ?", "101").First(&log).Error
	assert.NoError(t, err)
	assert.Equal(t, uint(1), log.UserID)
	assert.Equal(t, "transaction", log.Resource)
}

func BenchmarkLogTransaction(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		b.Fatalf("failed to connect database: %v", err)
	}

	db.AutoMigrate(&audit.AuditLog{})

	service := NewAuditService(db)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)
	c.Request.Header.Set("User-Agent", "Benchmark-Agent")
	c.Request.RemoteAddr = "192.168.1.1:1234"

	transaction := &models.Transaction{
		Model:           gorm.Model{ID: 1},
		AccountID:       1,
		Amount:          100.0,
		TransactionType: "deposit",
		Reference:       "Test deposit",
		Status:          "completed",
	}
	account := &models.Account{
		Model:         gorm.Model{ID: 1},
		UserID:        1,
		AccountNumber: "ACC123",
		Balance:       1000.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.LogTransaction(c, 1, transaction, account)
	}
}
