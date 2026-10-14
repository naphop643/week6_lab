package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

type Movie struct {
	ID      int64   `json:"id"`
	Title   string  `json:"title"`
	TitleTh *string `json:"titleTh"`
	Genre   *string `json:"genre"`
	Year    *int    `json:"year"`
	Rating  float64 `json:"rating"`
	Poster  *string `json:"poster"`
}

func (s *Store) SearchMovies(ctx context.Context, q string) ([]Movie, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT
			m.id,
			m.title,
			m.title_th,
			(
				SELECT COALESCE(g.name_th, g.name_en)
				FROM movie_genres mg
				JOIN genres g ON g.id = mg.genre_id
				WHERE mg.movie_id = m.id AND mg.position = 0
			),
			EXTRACT(YEAR FROM m.release_date)::int,
			ROUND(m.vote_average, 1)::float8,
			'https://image.tmdb.org/t/p/w500' || m.poster_path
		FROM movies m
		WHERE LOWER(m.title) LIKE '%' || LOWER($1) || '%'
		ORDER BY m.popularity DESC
	`, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movies []Movie

	for rows.Next() {
		var m Movie

		err := rows.Scan(
			&m.ID,
			&m.Title,
			&m.TitleTh,
			&m.Genre,
			&m.Year,
			&m.Rating,
			&m.Poster,
		)
		if err != nil {
			var pgErr *pgconn.PgError

			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				return nil, err
			}

			return nil, err
		}

		movies = append(movies, m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return movies, nil
}
