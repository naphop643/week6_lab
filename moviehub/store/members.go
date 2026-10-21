package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrEmailTaken = errors.New("email taken")

type Member struct {
	ID           int64  `json:"id"`
	Email        string `json:"email"`
	DisplayName  string `json:"displayName"`
	Role         string `json:"role"`
	PasswordHash string `json:"-"`
}

func (s *Store) CreateMember(
	ctx context.Context,
	email, hash, name string,
) (Member, error) {
	m := Member{
		Email:       email,
		DisplayName: name,
	}

	err := s.Pool.QueryRow(
		ctx,
		`INSERT INTO members (email, password_hash, display_name)
		 VALUES ($1, $2, $3)
		 RETURNING id, role`,
		email, hash, name,
	).Scan(&m.ID, &m.Role)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return Member{}, ErrEmailTaken
	}

	return m, err
}

func (s *Store) GetMemberByEmail(ctx context.Context, email string) (Member, error) {
	var m Member

	err := s.Pool.QueryRow(ctx, `
		SELECT id, email, display_name, role, password_hash
		FROM members
		WHERE lower(email) = lower($1)
	`, email).Scan(
		&m.ID,
		&m.Email,
		&m.DisplayName,
		&m.Role,
		&m.PasswordHash,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Member{}, ErrNotFound
		}
		return Member{}, err
	}

	return m, nil
}

func (s *Store) ListMembers(ctx context.Context) ([]Member, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, email, display_name, role
		FROM members
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]Member, 0)

	for rows.Next() {
		var m Member
		if err := rows.Scan(
			&m.ID,
			&m.Email,
			&m.DisplayName,
			&m.Role,
		); err != nil {
			return nil, err
		}
		members = append(members, m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return members, nil
}
