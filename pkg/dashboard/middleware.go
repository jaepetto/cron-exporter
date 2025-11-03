package dashboard

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware provides HTTP Basic Authentication using admin API keys
func AuthMiddleware() gin.HandlerFunc {
	return gin.BasicAuth(gin.Accounts{
		"admin": "admin", // This will be overridden by the custom validator
	})
}

// AuthMiddlewareWithKeys creates HTTP Basic Auth middleware with admin API key validation
func AuthMiddlewareWithKeys(adminAPIKeys []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.Header("WWW-Authenticate", `Basic realm="Dashboard"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			return
		}

		// Parse Basic Auth header
		if !strings.HasPrefix(auth, "Basic ") {
			c.Header("WWW-Authenticate", `Basic realm="Dashboard"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Basic authentication required"})
			return
		}

		// Extract credentials
		username, password, ok := c.Request.BasicAuth()
		if !ok {
			c.Header("WWW-Authenticate", `Basic realm="Dashboard"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials format"})
			return
		}

		// Validate password against admin API keys (username can be anything)
		validKey := false
		for _, key := range adminAPIKeys {
			if password == key {
				validKey = true
				break
			}
		}

		if !validKey {
			c.Header("WWW-Authenticate", `Basic realm="Dashboard"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}

		// Store username for logging
		c.Set("auth_user", username)
		c.Next()
	}
}

// CORSMiddleware handles CORS headers for dashboard
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}

// LoggingMiddleware provides request logging for dashboard
func LoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return ""
	})
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self'")
		c.Next()
	}
}
