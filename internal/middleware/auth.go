package middleware

import (
	"bank-api/internal/validation"
	"net/http"

	"bank-api/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AuthMiddleware struct {
	config *config.AppConfig
	secret []byte
}

func NewAuthMiddleware(cfg *config.AppConfig) *AuthMiddleware {
	return &AuthMiddleware{
		config: cfg,
		secret: []byte(cfg.JWT.Secret),
	}
}

func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}

		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Bearer token not found"})
			return
		}
		tokenString := authHeader[7:]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return m.secret, nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		// RFC 7519 defines 'sub' as a string, but many implementations use numbers.
		// Accommodate float64 and numeric strings (converting to uint for DB lookups),
		// while preserving non-numeric strings for external conventions.
		if subFloat, ok := claims["sub"].(float64); ok {
			c.Set("userID", uint(subFloat))
		} else if subStr, ok := claims["sub"].(string); ok {
			if id, err := validation.ParseUint(subStr, 0); err == nil {
				c.Set("userID", uint(id))
			} else {
				c.Set("userID", subStr)
			}
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
			return
		}

		c.Next()
	}
}
