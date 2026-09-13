package services

import (
	"bank-api/internal/audit"
	"bank-api/internal/models"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

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
