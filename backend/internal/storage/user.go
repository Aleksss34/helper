package storage

import (
	"context"
	"fmt"
)

func (p *Postgres) SetCountQuestions(ctx context.Context, userId int64, step int64) error {
	var op = "storage.user.SetCountQuestions"

	_, err := p.db.ExecContext(ctx, "UPDATE users SET count_questions = count_questions + $1 WHERE id = $2 RETURNING count_questions", step, userId)
	if err != nil {
		return fmt.Errorf("%s:%w", op, err)
	}
	return nil
}
