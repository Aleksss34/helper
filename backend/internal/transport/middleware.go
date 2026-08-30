package transport

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

func (t *Transport) CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")

		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getJwtToken(r *http.Request) (string, error) {
	// 1. Получаем заголовок Authorization
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header missing")
	}

	// 2. Формат заголовка должен быть: "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}

	// 3. Возвращаем чистый JWT
	return parts[1], nil
}

func (t *Transport) AuthMiddleware(next http.Handler) http.Handler {
	var op = "transport.middleware.Auth"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := getJwtToken(r)
		if err != nil {

			ctx := context.WithValue(r.Context(), "userID", int64(0))
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		id, err := t.auth.ParseJwtToken(tokenString)

		if err != nil {
			t.auth.log.Info("Cookie is expired or invalid", slog.String("op", op))
			ctx := context.WithValue(r.Context(), "userID", int64(0))
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		ctx := context.WithValue(r.Context(), "userID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
