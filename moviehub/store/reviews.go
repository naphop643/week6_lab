package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

type ReviewAuthor struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"displayName"`
}

type Review struct {
	ID        int64        `json:"id"`
	Text      string       `json:"text"`
	CreatedAt string       `json:"createdAt"`
	Author    ReviewAuthor `json:"author"`
	MyScore   *int         `json:"myScore"`
}

func (s *Store) ListReviews(
	ctx context.Context,
	movieID int64,
	memberID int64,
) ([]Review, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT
			r.id,
			r.text,
			r.created_at::text,
			m.id,
			m.display_name,
			v.score
		FROM reviews r
		JOIN members m ON m.id = r.member_id
		LEFT JOIN votes v
			ON v.movie_id = r.movie_id
			AND v.member_id = $2
		WHERE r.movie_id = $1
		ORDER BY r.created_at DESC
	`, movieID, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := make([]Review, 0)

	for rows.Next() {
		var r Review

		err := rows.Scan(
			&r.ID,
			&r.Text,
			&r.CreatedAt,
			&r.Author.ID,
			&r.Author.DisplayName,
			&r.MyScore,
		)
		if err != nil {
			return nil, err
		}

		reviews = append(reviews, r)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reviews, nil
}

func (s *Store) CreateReview(
	ctx context.Context,
	movieID int64,
	memberID int64,
	text string,
) (Review, error) {
	var r Review

	err := s.Pool.QueryRow(ctx, `
		INSERT INTO reviews (movie_id, member_id, text)
		VALUES ($1, $2, $3)
		RETURNING id, text, created_at::text
	`, movieID, memberID, text).Scan(
		&r.ID,
		&r.Text,
		&r.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return Review{}, ErrNotFound
		}

		return Review{}, err
	}

	err = s.Pool.QueryRow(ctx, `
		SELECT id, display_name
		FROM members
		WHERE id = $1
	`, memberID).Scan(
		&r.Author.ID,
		&r.Author.DisplayName,
	)
	if err != nil {
		return Review{}, err
	}

	return r, nil
}
