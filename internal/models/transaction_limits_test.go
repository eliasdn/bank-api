package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResetUsageIfNeeded(t *testing.T) {
	service := &TransactionLimitService{}

	t.Run("same day - no reset", func(t *testing.T) {
		now := time.Now()
		limit := &TransactionLimit{
			DailyUsed:     100,
			MonthlyUsed:   500,
			DepositUsed:   50,
			LastResetDate: now,
		}

		service.resetUsageIfNeeded(limit)

		assert.Equal(t, 100.0, limit.DailyUsed)
		assert.Equal(t, 500.0, limit.MonthlyUsed)
		assert.Equal(t, 50.0, limit.DepositUsed)
	})

	t.Run("new day same month - daily reset only", func(t *testing.T) {
		yesterday := time.Now().Add(-24 * time.Hour)
		limit := &TransactionLimit{
			DailyUsed:     100,
			MonthlyUsed:   500,
			DepositUsed:   50,
			LastResetDate: yesterday,
		}

		service.resetUsageIfNeeded(limit)

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
		limit := &TransactionLimit{
			DailyUsed:     100,
			MonthlyUsed:   500,
			DepositUsed:   50,
			LastResetDate: lastMonth,
		}

		service.resetUsageIfNeeded(limit)

		assert.Equal(t, 0.0, limit.DailyUsed)
		assert.Equal(t, 0.0, limit.DepositUsed)
		assert.Equal(t, 0.0, limit.MonthlyUsed)
	})
}
