package middleware

import (
	"context"

	"net/http"
	"os"
	"strings"

	kafkalogger "github.com/Explorerr/pet_project/pkg/Kafka_logger"
	models "github.com/Explorerr/pet_project/pkg/Models"
	contextkey "github.com/Explorerr/pet_project/pkg/context_Key"

	"github.com/golang-jwt/jwt/v5"
)

type StatusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *StatusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

func CORS_Middleware(next http.Handler, log kafkalogger.LoggerInterface) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization,X-Forwarded-For")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent) // 204 No Content
			return
		}
		next.ServeHTTP(w, r)
	})
}

func Info_Middleware(next http.Handler, log kafkalogger.LoggerInterface) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ip_address := r.Header.Get("X-Forwarded-For")
		userAgent := r.Header.Get("User-Agent")

		req_info := models.Request_Info{Ip_add: ip_address, Source: userAgent, Method: r.Method, Path: r.URL.Path}
		ctx := context.WithValue(r.Context(), contextkey.ReqInfo, req_info)
		next.ServeHTTP(w, r.WithContext(ctx))

	})
}

func JWT_Middleware(next http.Handler, log kafkalogger.LoggerInterface) http.Handler {
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

		ctx := context.WithValue(r.Context(), contextkey.UserID, claims.UserID)
		ctx = context.WithValue(ctx, contextkey.UserRole, claims.Role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
