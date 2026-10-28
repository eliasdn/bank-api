package middleware

import (
	"bank-api/internal/config"
	"bank-api/internal/validation"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	authErrorMissingHeaderJSON      = []byte(`{"error":"Authorization header is required"}`)
	authErrorBearerNotFoundJSON     = []byte(`{"error":"Bearer token not found"}`)
	authErrorInvalidTokenJSON       = []byte(`{"error":"Invalid token"}`)
	authErrorInvalidTokenClaimsJSON = []byte(`{"error":"Invalid token claims"}`)
	authErrorInvalidUserIDJSON      = []byte(`{"error":"Invalid user ID in token"}`)
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
			c.Data(http.StatusUnauthorized, "application/json", authErrorMissingHeaderJSON)
			c.Abort()
			return
		}

		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			c.Data(http.StatusUnauthorized, "application/json", authErrorBearerNotFoundJSON)
			c.Abort()
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
			c.Data(http.StatusUnauthorized, "application/json", authErrorInvalidTokenJSON)
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.Data(http.StatusUnauthorized, "application/json", authErrorInvalidTokenClaimsJSON)
			c.Abort()
			return
		}

		// RFC 7519 defines 'sub' as a string, but many implementations use numbers.
		// Accommodate float64, integers, and numeric strings (converting to uint for DB lookups),
		// while validating bounds (>0, finite, integer) to prevent underflow/wrap-around.
		switch v := claims["sub"].(type) {
		case float64:
			if v > 0 && !math.IsInf(v, 0) && v == math.Trunc(v) {
				c.Set("userID", uint(v))
			} else {
				c.Data(http.StatusUnauthorized, "application/json", authErrorInvalidUserIDJSON)
				c.Abort()
				return
			}
		case string:
			if id, err := validation.ParseUint(v, 0); err == nil && id > 0 {
				c.Set("userID", uint(id))
			} else if v != "" && v != "0" {
				c.Set("userID", v)
			} else {
				c.Data(http.StatusUnauthorized, "application/json", authErrorInvalidUserIDJSON)
				c.Abort()
				return
			}
		case uint:
			if v > 0 {
				c.Set("userID", v)
			} else {
				c.Data(http.StatusUnauthorized, "application/json", authErrorInvalidUserIDJSON)
				c.Abort()
				return
			}
		case uint64:
			if v > 0 {
				c.Set("userID", uint(v))
			} else {
				c.Data(http.StatusUnauthorized, "application/json", authErrorInvalidUserIDJSON)
				c.Abort()
				return
			}
		case int:
			if v > 0 {
				c.Set("userID", uint(v))
			} else {
				c.Data(http.StatusUnauthorized, "application/json", authErrorInvalidUserIDJSON)
				c.Abort()
				return
			}
		case int64:
			if v > 0 {
				c.Set("userID", uint(v))
			} else {
				c.Data(http.StatusUnauthorized, "application/json", authErrorInvalidUserIDJSON)
				c.Abort()
				return
			}
		default:
			c.Data(http.StatusUnauthorized, "application/json", authErrorInvalidUserIDJSON)
			c.Abort()
			return
		}

		c.Next()
	}
}
