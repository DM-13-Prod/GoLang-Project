package repository

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq"
	"todo/internal/model"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(connStr string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Load() ([]model.TaskDTO, error) {
	rows, err := s.db.Query(`SELECT id, title, description, status, priority, due_at, created_at, updated_at, completed_at FROM tasks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.TaskDTO
	for rows.Next() {
		var r model.TaskDTO
		rows.Scan(&r.ID, &r.Title, &r.Description, &r.Status, &r.Priority,
			&r.DueAt, &r.CreatedAt, &r.UpdatedAt, &r.CompletedAt)
		items = append(items, r)
	}
	return items, nil
}

func (s *PostgresStore) Save(items []model.TaskDTO) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`TRUNCATE TABLE tasks RESTART IDENTITY`); err != nil {
		return err
	}
	stmt, err := tx.Prepare(`
		INSERT INTO tasks (id, title, description, status, priority, due_at, created_at, updated_at, completed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, t := range items {
		_, err := stmt.Exec(t.ID, t.Title, t.Description, t.Status, t.Priority,
			t.DueAt, t.CreatedAt, t.UpdatedAt, t.CompletedAt)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}