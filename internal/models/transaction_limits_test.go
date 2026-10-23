package models_test

import (
	"bank-api/internal/models"
	"bank-api/internal/testutils"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResetUsageIfNeeded(t *testing.T) {
	service := models.NewTransactionLimitService(nil)

	t.Run("same day - no reset", func(t *testing.T) {
		now := time.Now()
		limit := &models.TransactionLimit{
			DailyUsed:     100,
			MonthlyUsed:   500,
			DepositUsed:   50,
			LastResetDate: now,
		}

		models.ResetUsageIfNeededForTest(service, limit)

		assert.Equal(t, 100.0, limit.DailyUsed)
		assert.Equal(t, 500.0, limit.MonthlyUsed)
		assert.Equal(t, 50.0, limit.DepositUsed)
	})

	t.Run("new day same month - daily reset only", func(t *testing.T) {
		yesterday := time.Now().Add(-24 * time.Hour)
		limit := &models.TransactionLimit{
			DailyUsed:     100,
			MonthlyUsed:   500,
			DepositUsed:   50,
			LastResetDate: yesterday,
		}

		models.ResetUsageIfNeededForTest(service, limit)

		assert.Equal(t, 0.0, limit.DailyUsed)
		assert.Equal(t, 0.0, limit.DepositUsed)
		// If yesterday was in the same month, monthly used remains
		if yesterday.Month() == time.Now().Month() && yesterday.Year() == time.Now().Year() {
			assert.Equal(t, 500.0, limit.MonthlyUsed)
		} else {
			assert.Equal(t, 0.0, limit.MonthlyUsed)
		}
	})

	t.Run("new month - both daily and monthly reset", func(t *testing.T) {
		// Set LastResetDate to 35 days ago (different month)
		lastMonth := time.Now().Add(-35 * 24 * time.Hour)
		limit := &models.TransactionLimit{
			DailyUsed:     100,
			MonthlyUsed:   500,
			DepositUsed:   50,
			LastResetDate: lastMonth,
		}

		models.ResetUsageIfNeededForTest(service, limit)

		assert.Equal(t, 0.0, limit.DailyUsed)
		assert.Equal(t, 0.0, limit.DepositUsed)
		assert.Equal(t, 0.0, limit.MonthlyUsed)
	})
}

func TestRecordTransaction_Concurrent(t *testing.T) {
	db := testutils.InitTestDB()
	service := models.NewTransactionLimitService(db)

	accountID := uint(100)
	concurrency := 10
	amountPerTx := 50.0

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		txID := uint(i + 1)
		go func(id uint) {
			defer wg.Done()
			err := service.RecordTransaction(accountID, id, amountPerTx, "deposit")
			assert.NoError(t, err)
		}(txID)
	}

	wg.Wait()

	limit, err := service.GetLimit(accountID)
	require.NoError(t, err)

	expectedTotal := float64(concurrency) * amountPerTx
	assert.Equal(t, expectedTotal, limit.DailyUsed)
	assert.Equal(t, expectedTotal, limit.MonthlyUsed)
	assert.Equal(t, expectedTotal, limit.DepositUsed)
}
