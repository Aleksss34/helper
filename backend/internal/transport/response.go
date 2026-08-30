package transport

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Aleksss34/helper/backend/internal/domain"
	"github.com/Aleksss34/helper/backend/internal/dto"
)

func mapErrorToREST(err error) *dto.APIError {
	switch {
	// Конфликты занятых данных (409 Conflict)
	case errors.Is(err, domain.ErrUsernameExists), errors.Is(err, domain.ErrUserExists):
		return withFieldError(http.StatusConflict, "Username already exists", "username", "USERNAME_ALREADY_EXISTS")

	case errors.Is(err, domain.ErrEmailExists):
		return withFieldError(http.StatusConflict, "Email already exists", "email", "EMAIL_ALREADY_EXISTS")

	// Некорректные аргументы (400 Bad Request)
	case errors.Is(err, domain.ErrPassSimple):
		return withFieldError(http.StatusBadRequest, "Validation failed", "password", "WEAK_PASSWORD")

	// Неверные учётные данные (401) — намеренно один и тот же код для обоих случаев,
	// чтобы не раскрывать, существует ли username (защита от enumeration)
	case errors.Is(err, domain.ErrInvalidUsername), errors.Is(err, domain.ErrInvalidPass):
		return &dto.APIError{Status: http.StatusUnauthorized, Message: "Invalid credentials", Code: "INVALID_CREDENTIALS"}

	case errors.Is(err, domain.ErrExpiredRefreshToken):
		return &dto.APIError{Status: http.StatusUnauthorized, Message: err.Error(), Code: "SESSION_EXPIRED"}

	case errors.Is(err, domain.ErrLimitReached):
		return &dto.APIError{Status: http.StatusTooManyRequests, Message: err.Error(), Code: "LIMIT_REACHED"}

	default:
		return &dto.APIError{Status: http.StatusInternalServerError, Message: "internal server error", Code: "INTERNAL_ERROR"}
	}
}

func withFieldError(status int, message, field, code string) *dto.APIError {
	return &dto.APIError{
		Status:  status,
		Message: message,
		Code:    code,
		Field: &dto.FieldError{
			Field:   field,
			Code:    code,
			Message: message,
		},
	}
}
func writeError(w http.ResponseWriter, err error) {
	apiErr := mapErrorToREST(err)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.Status)
	json.NewEncoder(w).Encode(apiErr)
}

func respondJSON(w http.ResponseWriter, log *slog.Logger, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {

		log.Error("Не удалось отправить json", slog.Any("error", err))
	}
}
