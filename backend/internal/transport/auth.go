package transport

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Aleksss34/helper/backend/internal/dto"
	"github.com/golang-jwt/jwt/v5"
)

func (a *Auth) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, &dto.APIError{Status: http.StatusBadRequest, Message: "invalid request body"})
		return
	}

	refreshToken, accessToken, err := a.serv.RegisterService(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		//Secure:   true,              // только по HTTPS
		SameSite: http.SameSiteStrictMode,
		Path:     "/api/auth/refresh",
		MaxAge:   60 * 60 * 24 * 30,
	})
	respondJSON(w, a.log, http.StatusOK, map[string]string{
		"access_token": accessToken,
	})
}

func (a *Auth) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, &dto.APIError{Status: http.StatusBadRequest, Message: "invalid request body"})
		return
	}

	refreshToken, accessToken, err := a.serv.LoginService(r.Context(), req.Username, req.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		//Secure:   true,              // только по HTTPS
		SameSite: http.SameSiteStrictMode,
		Path:     "/api/refresh",
		MaxAge:   60 * 60 * 24 * 30,
	})
	respondJSON(w, a.log, http.StatusOK, map[string]string{
		"access_token": accessToken,
	})
}
func (a *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")

	if err != nil {
		writeError(w, &dto.APIError{Status: http.StatusBadRequest, Message: "refresh token not found"})
		return
	}

	if err := a.serv.LogoutService(r.Context(), cookie.Value); err != nil {

		writeError(w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HttpOnly: true,
		//Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/api/refresh",
		MaxAge:   -1, // немедленно удаляет cookie
	})

	respondJSON(w, a.log, http.StatusOK, map[string]string{
		"message": "logged out",
	})
}
func (a *Auth) Refresh(w http.ResponseWriter, r *http.Request) {
	var op = "transport.auth.Refresh"
	log := a.log.With(slog.String("op", op))
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		// Если куки нет — это 401 Unauthorized (пользователь не залогинен), а не 500
		log.Info("Пользователь не авторизован", slog.Any("error", err))
		writeError(w, &dto.APIError{Status: http.StatusUnauthorized, Message: "refresh token not found"})

		return
	}

	// 2. Передаем значение из cookie.Value в сервис
	refreshToken, accessToken, err := a.serv.RefreshService(r.Context(), cookie.Value)
	if err != nil {
		log.Error("Не удалось получить токены", slog.Any("error", err))
		writeError(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		//Secure:   true,              // только по HTTPS
		SameSite: http.SameSiteStrictMode,
		Path:     "/api/refresh",
		MaxAge:   60 * 60 * 24 * 30,
	})
	respondJSON(w, a.log, http.StatusOK, map[string]string{
		"access_token": accessToken,
	})
}

func (a *Auth) MeHandler(w http.ResponseWriter, r *http.Request) {

	user, err := a.serv.MeService(r.Context())
	if err != nil {

		writeError(w, err)
		return
	}
	fmt.Println(user)
	resp := dto.UserResponse{ID: user.Id, Email: user.Email, Status: user.Status, Username: user.Username, CountQuestions: user.CountQuestions}
	respondJSON(w, a.log, http.StatusOK, resp)
}

func (a *Auth) IsAdmin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userId := ctx.Value("userID")
	fmt.Println(userId, "chita")
	ok, err := a.serv.IsAdminService(ctx, userId.(int64))
	if err != nil {
		writeError(w, err)
		return
	}
	if ok {
		respondJSON(w, a.log, http.StatusOK, map[string]string{
			"status": "true",
		})
	} else {
		respondJSON(w, a.log, http.StatusOK, map[string]string{
			"status": "false",
		})
	}

}

func (a *Auth) ParseJwtToken(tokenString string) (int64, error) {
	var op = "service.auth,ParseJwtToken"
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			a.log.Warn("suspicious request: invalid token signature algorithm", "actual_alg", token.Header["alg"])
			return "", fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.hmacSecret), nil
	})

	if err != nil {

		a.log.Warn("error parsing JWT token", "error", err)
		return 0, err
	}
	if !token.Valid {
		a.log.Warn("invalid JWT token")
		return 0, fmt.Errorf("%s: invalid jwt token", op)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		a.log.Warn("не удалось прочитать claims из JWT токена")
		return 0, fmt.Errorf("%s: invalid claims", op)
	}

	subject, ok := claims["sub"].(string)
	if !ok {
		a.log.Warn("invalid sub in JWT token")
		return 0, fmt.Errorf("%s: invalid sub", op)
	}

	id, err := strconv.Atoi(subject)
	if err != nil {
		slog.Warn("failed converting sub JWT in int64", "sub", subject)
		return 0, fmt.Errorf("%s: invalid converting sub", op)
	}

	return int64(id), nil
}
