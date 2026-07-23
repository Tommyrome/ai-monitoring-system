package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func jwtSecret() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "dev-secret-change-me"
	}
	return []byte(s)
}

// GenerateToken crea un JWT firmato per l'utente autenticato dalla dashboard.
func GenerateToken(userID, username, ruolo string) (string, error) {
	claims := jwt.MapClaims{
		"sub":      userID,
		"username": username,
		"ruolo":    ruolo,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}

// RequireJWT protegge le route della dashboard (utenti autenticati).
func RequireJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token mancante"})
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return jwtSecret(), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token non valido"})
			return
		}
		c.Next()
	}
}

// RequireServiceToken protegge l'endpoint usato dal modulo AI Python
// (un semplice bearer token condiviso, piu' leggero di un JWT per servizio-a-servizio).
func RequireServiceToken() gin.HandlerFunc {
	expected := os.Getenv("AI_SERVICE_TOKEN")
	if expected == "" {
		expected = "change-me-shared-secret"
	}
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header != "Bearer "+expected {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token servizio non valido"})
			return
		}
		c.Next()
	}
}
