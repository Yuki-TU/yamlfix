package example

import (
	"context"
	"database/sql"
	"time"
)

type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

var (
	_ Queryer = (*sql.DB)(nil)
	_ Execer  = (*sql.DB)(nil)
	_ Execer  = (*sql.Tx)(nil)
)

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

type User struct {
	ID        uint64
	Name      string
	Email     string
	CreatedAt time.Time
}

func (r *Repository) CreateUser(ctx context.Context, db Execer, n User) (User, error) {
	sql := `
		INSERT INTO users (
			name,
			email,
			created_at
		)
		VALUES (?,?,?)
	`
	result, err := db.ExecContext(ctx, sql, n.Name, n.Email, time.Now())
	if err != nil {
		return n, err
	}
	id, _ := result.LastInsertId()
	n.ID = uint64(id)
	return n, nil
}

func (r *Repository) GetUser(ctx context.Context, db Queryer, id uint64) (User, error) {
	sql := `
		SELECT id, name, email, created_at
		FROM users
		WHERE id = ?
	`
	rows, err := db.QueryContext(ctx, sql, id)
	if err != nil {
		return User{}, err
	}
	defer rows.Close()

	var u User
	if rows.Next() {
		err = rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
		if err != nil {
			return User{}, err
		}
	}
	return u, nil
}
