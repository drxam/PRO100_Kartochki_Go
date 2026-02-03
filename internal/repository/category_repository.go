package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pro100kartochki/mozgoemka/internal/domain"
)

type CategoryRepository struct {
	db *DB
}

func NewCategoryRepository(db *DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) Create(ctx context.Context, c *domain.Category) error {
	query := `INSERT INTO categories (name) VALUES ($1) RETURNING id, created_at`
	return r.db.Pool.QueryRow(ctx, query, c.Name).Scan(&c.ID, &c.CreatedAt)
}

func (r *CategoryRepository) GetByID(ctx context.Context, id int) (*domain.Category, error) {
	query := `SELECT id, name, created_at FROM categories WHERE id = $1`
	var c domain.Category
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(&c.ID, &c.Name, &c.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *CategoryRepository) GetByName(ctx context.Context, name string) (*domain.Category, error) {
	query := `SELECT id, name, created_at FROM categories WHERE name = $1`
	var c domain.Category
	err := r.db.Pool.QueryRow(ctx, query, name).Scan(&c.ID, &c.Name, &c.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *CategoryRepository) List(ctx context.Context) ([]domain.Category, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT id, name, created_at FROM categories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.Category
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}
