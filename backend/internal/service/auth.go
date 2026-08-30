package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/Aleksss34/helper/backend/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

func (a *Auth) RegisterService(ctx context.Context, username, email, pass string) (string, string, error) {
	const op = "service.Auth.Register"
	log := a.log.With(slog.String("op", op))
	if err := a.validatePass(username, pass); err != nil {
		log.Info("Невалидный пароль", slog.Any("error", err))
		return "", "", fmt.Errorf("%s:%w", op, err)
	}
	if err := ctx.Err(); err != nil {
		log.Info("Запрос отменен контекстом до хеширования", slog.Any("error", err))
		return "", "", fmt.Errorf("%s: %w", op, err)
	}
	hashPass, err := a.hasher.Hash(pass)
	if err != nil {
		log.Error("Не удалось захешировать пароль", slog.Any("error", err))
		return "", "", fmt.Errorf("%s: %w", op, err)
	}
	id, err := a.postgres.AddUser(ctx, username, email, hashPass)

	if err != nil {
		if errors.Is(err, domain.ErrUserExists) {
			log.Info("Данный пользователь уже зарегестрирован", slog.Any("error", err))
		} else if errors.Is(err, domain.ErrEmailExists) {
			log.Info("Данная почта уже зарегестрирована", slog.Any("error", err))
		} else if errors.Is(err, domain.ErrUsernameExists) {
			log.Info("Данный username уже зарегестрирован", slog.Any("error", err))
		} else {
			log.Error("Не удалось добавить пользователя в базу данных", slog.Any("error", err))
		}

		return "", "", fmt.Errorf("%s:%w", op, err)
	}

	accessToken, err := a.generateAccessToken(id)
	if err != nil {
		log.Error("Не удалось сгенерировать jwt токен", slog.Any("error", err))
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	plain, hash, err := a.generateRefreshToken()
	if err != nil {
		log.Error("Не удалось сгенерировать refresh токен", slog.Any("error", err))
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	if err := a.redis.AddRefresh(ctx, hash, id); err != nil {
		log.Error("Не удалось добавить токен в redis", slog.Any("error", err))
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	return plain, accessToken, nil
}

func (a *Auth) LoginService(ctx context.Context, username, pass string) (string, string, error) {
	var op = "service.auth.Login"
	log := a.log.With(slog.String("op", op))
	if err := ctx.Err(); err != nil {
		log.Info("Контекст закрыт до поиска в бд", slog.Any("error", err))
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	user, err := a.postgres.GetUser(ctx, username)
	if err != nil {
		a.hasher.Compare(pass, a.dummyHash)
		log.Error("Не удалось найти пользователя в бд", slog.Any("error", err))
		return "", "", fmt.Errorf("%s:%w", op, domain.ErrInvalidUsername)
	}
	if err := ctx.Err(); err != nil {
		log.Info("Контекст закрыт до хеширования", slog.Any("error", err))
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	if !a.hasher.Compare(pass, user.HashPass) {
		log.Info("Неправильный пароль")
		return "", "", fmt.Errorf("%s:%w", op, domain.ErrInvalidPass)
	}

	if err = ctx.Err(); err != nil {
		log.Info("Контекст закрыт до генерации jwt токена", slog.Any("error", err))
		return "", "", fmt.Errorf("%s: %w", op, err)
	}
	token, err := a.generateAccessToken(user.Id)
	if err != nil {
		log.Error("Не удалось сгенерировать jwt токен", slog.Any("error", err))
		return "", "", fmt.Errorf("%s:%w", op, err)
	}
	plain, hash, err := a.generateRefreshToken()
	if err != nil {
		log.Error("Не удалось сгенерировать refresh токен", slog.Any("error", err))
		return "", "", fmt.Errorf("%s:%w", op, err)
	}
	if err = a.redis.AddRefresh(ctx, hash, user.Id); err != nil {
		log.Error("Не удалось добавить токен в redis")
		return "", "", fmt.Errorf("%s:%w", op, err)
	}
	return plain, token, nil

}
func (a *Auth) LogoutService(ctx context.Context, refreshToken string) error {
	return a.redis.DelRefresh(ctx, refreshToken)
}
func (a *Auth) IsAdminService(ctx context.Context, userId int64) (bool, error) {
	var op = "service.auth.IsAdminService"
	user, err := a.postgres.GetUserByID(ctx, userId)
	if err != nil {
		return false, fmt.Errorf("%s:%w", op, err)
	}
	if user.Status == "admin" {
		return true, nil
	}
	return false, nil

}

func (a *Auth) MeService(ctx context.Context) (*domain.User, error) {
	var op = "service.auth.MeService"
	userId := ctx.Value("userID").(int64)
	user, err := a.postgres.GetUserByID(ctx, userId)
	if err != nil {
		a.log.Error("Не удалось найти пользователя по айди", slog.Any("error", err), slog.String("op", op))
		return nil, fmt.Errorf("%s:%w", op, err)
	}
	return &user, nil
}

func (a *Auth) validatePass(username, pass string) error {
	var op = "service.auth.validatePass"
	if len(username) <= 3 {
		return nil
	}
	pass, username = strings.ToLower(pass), strings.ToLower(username)
	if strings.Contains(pass, username) {
		return fmt.Errorf("%s: %w", op, domain.ErrPassSimple)
	}
	return nil
}

func (a *Auth) RefreshService(ctx context.Context, refreshToken string) (string, string, error) {
	const op = "service.auth.RefreshService"
	log := a.log.With(slog.String("op", op))

	if err := ctx.Err(); err != nil {
		log.Info("Контекст закрыт до рефреша токена", slog.Any("error", err))
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	h := sha256.Sum256([]byte(refreshToken))
	hash := hex.EncodeToString(h[:])

	id, err := a.redis.GetIdByRefresh(ctx, hash)
	if err != nil {

		log.Warn("Рефреш токен не найден — истёк или переиспользован", slog.Any("error", err))
		return "", "", fmt.Errorf("%s: %w", op, domain.ErrExpiredRefreshToken)
	}

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Error("Айди не целое число", slog.Any("error", err))
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	if err := a.redis.DelRefresh(ctx, hash); err != nil {
		log.Error("Не удалось удалить старый refresh токен", slog.Any("error", err))
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	accessToken, err := a.generateAccessToken(idInt)
	if err != nil {
		log.Error("Не удалось сгенерировать access токен", slog.Any("error", err))
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	newPlain, newHash, err := a.generateRefreshToken()
	if err != nil {
		log.Error("Не удалось сгенерировать новый refresh токен", slog.Any("error", err))
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	if err := a.redis.AddRefresh(ctx, newHash, idInt); err != nil {
		log.Error("Не удалось сохранить новый refresh токен", slog.Any("error", err))
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	return newPlain, accessToken, nil
}

func (a *Auth) generateAccessToken(userId int64) (string, error) {
	var op = "service.auth.generateAccessToken"
	mapClaims := jwt.MapClaims{
		"sub": strconv.FormatInt(userId, 10),
		"exp": time.Now().Add(30 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, mapClaims)
	tokenString, err := token.SignedString([]byte(a.hmacSecret))
	if err != nil {
		a.log.Error("Не удалось подписать токен", slog.Any("error", err), slog.String("op", op))
		return "", fmt.Errorf("%s:%w", op, err)
	}
	return tokenString, nil
}

func (a *Auth) generateRefreshToken() (string, string, error) {
	var op = "service.auth.generateRefreshToken"
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("%s:%w", op, err)
	}
	plain := base64.URLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(plain))
	hash := hex.EncodeToString(h[:])
	return plain, hash, nil
}
