package middleware

import (
	"context"

	"net/http"
	"os"
	"strings"

	kafkalogger "github.com/Explorerr/pet_project/pkg/Kafka_logger"
	models "github.com/Explorerr/pet_project/pkg/Models"

	"github.com/golang-jwt/jwt/v5"
)

func JWT_Middleware(next http.Handler, log *kafkalogger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.ERROR("Middleware", "middleware_auth", "Missing Authorization header", nil)
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.ERROR("Middleware", "middlewarer_auth", "Invalid Authorization header format", nil)
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}
		tokenString := parts[1]

		JWTSecret := os.Getenv("JWT_SECRET")
		if JWTSecret == "" {
			log.ERROR("Middleware", "middlewarer_env", "JWT_SECRET not set in environment", nil)

			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		token, err := jwt.ParseWithClaims(tokenString, &models.MyClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(JWTSecret), nil
		})
		if err != nil || !token.Valid {
			log.ERROR("Middleware", "middleware_jwt", "Invalid or expired token", nil)
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}
		claims, ok := token.Claims.(*models.MyClaims)
		if !ok {
			log.ERROR("Middleware", "middleware_jwt", "Invalid token claims", nil)
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "user_role", claims.Role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
