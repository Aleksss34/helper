package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/Aleksss34/helper/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

func (p *Postgres) AddUser(ctx context.Context, username, email, hashPass string) (int64, error) {
	var op = "storage.auth.add"
	var id int64

	err := p.db.QueryRowContext(ctx, "INSERT INTO users(username, email, hash_password, status, count_questions) VALUES($1, $2, $3, $4, $5) RETURNING id", username, email, hashPass, "base", 15).Scan(&id)
	if err != nil {

		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "idx_users_username", "users_username_key":
				return 0, fmt.Errorf("%s:%w", op, domain.ErrUsernameExists)
			case "idx_users_email", "users_email_key":
				return 0, fmt.Errorf("%s:%w", op, domain.ErrEmailExists)
			default:
				return 0, fmt.Errorf("%s:%w", op, domain.ErrUserExists)
			}
		}

		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (p *Postgres) GetUser(ctx context.Context, username string) (domain.User, error) {
	var op = "storage.auth.Get"
	var user domain.User

	if err := p.db.QueryRowContext(ctx, "SELECT * FROM users WHERE username=$1", username).Scan(&user.Id, &user.Username, &user.Email, &user.HashPass, &user.Status, &user.CountQuestions); err != nil {

		return domain.User{}, fmt.Errorf("%s:%w", op, err)
	}
	return user, nil

}

func (p *Postgres) GetUserByID(ctx context.Context, userId int64) (domain.User, error) {
	var op = "storage.auth.Get"
	var user domain.User

	if err := p.db.QueryRowContext(ctx, "SELECT * FROM users WHERE id=$1", userId).Scan(&user.Id, &user.Username, &user.Email, &user.HashPass, &user.Status, &user.CountQuestions); err != nil {

		return domain.User{}, fmt.Errorf("%s:%w", op, err)
	}
	return user, nil

}
