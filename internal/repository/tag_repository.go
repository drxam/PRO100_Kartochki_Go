package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
)

type TagRepository struct {
	db *DB
}

func NewTagRepository(db *DB) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) Create(ctx context.Context, t *domain.Tag) error {
	query := `INSERT INTO tags (name) VALUES ($1) RETURNING id, created_at`
	return r.db.Pool.QueryRow(ctx, query, t.Name).Scan(&t.ID, &t.CreatedAt)
}

func (r *TagRepository) GetByID(ctx context.Context, id int) (*domain.Tag, error) {
	query := `SELECT id, name, created_at FROM tags WHERE id = $1`
	var t domain.Tag
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(&t.ID, &t.Name, &t.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *TagRepository) GetByName(ctx context.Context, name string) (*domain.Tag, error) {
	query := `SELECT id, name, created_at FROM tags WHERE name = $1`
	var t domain.Tag
	err := r.db.Pool.QueryRow(ctx, query, name).Scan(&t.ID, &t.Name, &t.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *TagRepository) GetByIDs(ctx context.Context, ids []int) ([]domain.Tag, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.db.Pool.Query(ctx, `SELECT id, name, created_at FROM tags WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.Tag
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func (r *TagRepository) List(ctx context.Context) ([]domain.Tag, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT id, name, created_at FROM tags ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.Tag
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func (r *TagRepository) Update(ctx context.Context, t *domain.Tag) error {
	query := `UPDATE tags SET name=$2 WHERE id=$1 RETURNING id`
	var id int
	return r.db.Pool.QueryRow(ctx, query, t.ID, t.Name).Scan(&id)
}

func (r *TagRepository) Delete(ctx context.Context, id int) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM tags WHERE id = $1`, id)
	return err
}

// ListWithSearch возвращает теги с опциональным поиском по имени.
func (r *TagRepository) ListWithSearch(ctx context.Context, search string) ([]domain.Tag, error) {
	if search == "" {
		return r.List(ctx)
	}
	rows, err := r.db.Pool.Query(ctx, `SELECT id, name, created_at FROM tags WHERE name ILIKE $1 ORDER BY name`, "%"+search+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.Tag
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}
