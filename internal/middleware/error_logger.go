package middleware

import (
	"bank-api/internal/config"
	"bank-api/internal/errors"
	"bank-api/internal/metrics"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ErrorLogger struct {
	config *config.AppConfig
}

func NewErrorLogger(cfg *config.AppConfig) *ErrorLogger {
	return &ErrorLogger{
		config: cfg,
	}
}

func (m *ErrorLogger) LogErrors() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Generate or get request ID
		requestID := c.GetString("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
			c.Set("X-Request-ID", requestID)
		}

		c.Next()

		if len(c.Errors) > 0 {
			for _, ginErr := range c.Errors {
				err := ginErr.Err
				if appErr, ok := err.(*errors.Error); ok {
					log.Printf("[ERROR] request_id:%s %s %s - code:%s message:%s status:%d latency:%v error:%v",
						requestID,
						c.Request.Method,
						c.Request.URL.Path,
						appErr.Code,
						appErr.Message,
						appErr.Status,
						time.Since(start),
						appErr.Err,
					)
					metrics.ErrorCounter.WithLabelValues(appErr.Code).Inc()
				} else {
					log.Printf("[ERROR] request_id:%s %s %s - latency:%v error:%v",
						requestID,
						c.Request.Method,
						c.Request.URL.Path,
						time.Since(start),
						err,
					)
					metrics.ErrorCounter.WithLabelValues("unknown").Inc()
				}
			}
		}
	}
}
