package store

import (
	"context"
	"errors"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Menu struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Type  string  `json:"type"`
	Stock int     `json:"stock"`
}

type TopMenu struct {
	Name string `json:"name"`
	Jan  int64  `json:"jan"`
	Baht int64  `json:"baht"`
}

var ErrNotFound = errors.New("menu not found")
var ErrInUse = errors.New("menu in use")

type MenuStore interface {
	List(ctx context.Context, menuType string) ([]Menu, error)
	Get(ctx context.Context, id int) (Menu, error)
	Add(ctx context.Context, m Menu) (Menu, error)
	Delete(ctx context.Context, id int) error
	TopMenus(ctx context.Context, limit int) ([]TopMenu, error)
}

// =========================
// MemoryStore
// =========================

type MemoryStore struct {
	mu     sync.RWMutex
	menus  []Menu
	nextID int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		menus: []Menu{
			{ID: 1, Name: "ต้มยำกุ้ง", Price: 120, Type: "soup"},
			{ID: 2, Name: "พิซซ่าฮาวายเอี้ยน", Price: 199, Type: "pizza"},
			{ID: 3, Name: "พิซซ่าเห็ด", Price: 179, Type: "pizza"},
		},
		nextID: 4,
	}
}

func (s *MemoryStore) List(ctx context.Context, menuType string) ([]Menu, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if menuType == "" {
		out := make([]Menu, len(s.menus))
		copy(out, s.menus)
		return out, nil
	}

	filtered := []Menu{}

	for _, m := range s.menus {
		if m.Type == menuType {
			filtered = append(filtered, m)
		}
	}

	return filtered, nil
}

func (s *MemoryStore) Get(ctx context.Context, id int) (Menu, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, m := range s.menus {
		if m.ID == id {
			return m, nil
		}
	}

	return Menu{}, ErrNotFound
}

func (s *MemoryStore) Add(ctx context.Context, m Menu) (Menu, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m.ID = s.nextID
	s.nextID++

	s.menus = append(s.menus, m)

	return m, nil
}

func (s *MemoryStore) Delete(ctx context.Context, id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, m := range s.menus {
		if m.ID == id {
			s.menus = append(s.menus[:i], s.menus[i+1:]...)
			return nil
		}
	}

	return ErrNotFound
}

// =========================
// PostgresStore
// =========================

type PostgresStore struct {
	Pool *pgxpool.Pool
}

func (s *PostgresStore) List(ctx context.Context, menuType string) ([]Menu, error) {
	rows, err := s.Pool.Query(
		ctx,
		`SELECT id, name, price, type, stock
		 FROM menus
		 WHERE ($1 = '' OR type = $1)
		 ORDER BY id`,
		menuType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	menus := []Menu{}

	for rows.Next() {
		var m Menu

		if err := rows.Scan(
			&m.ID,
			&m.Name,
			&m.Price,
			&m.Type,
			&m.Stock,
		); err != nil {
			return nil, err
		}

		menus = append(menus, m)
	}

	return menus, rows.Err()
}

func (s *PostgresStore) Get(ctx context.Context, id int) (Menu, error) {
	var m Menu

	err := s.Pool.QueryRow(
		ctx,
		`SELECT id, name, price, type, stock
		 FROM menus
		 WHERE id = $1`,
		id,
	).Scan(
		&m.ID,
		&m.Name,
		&m.Price,
		&m.Type,
		&m.Stock,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return Menu{}, ErrNotFound
	}

	return m, err
}

// =========================
// Add Menu
// =========================

func (s *PostgresStore) Add(ctx context.Context, m Menu) (Menu, error) {
	err := s.Pool.QueryRow(
		ctx,
		`INSERT INTO menus (name, price, type, stock)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		m.Name,
		m.Price,
		m.Type,
		m.Stock,
	).Scan(&m.ID)

	return m, err
}

// =========================
// Delete Menu
// =========================

func (s *PostgresStore) Delete(ctx context.Context, id int) error {
	tag, err := s.Pool.Exec(
		ctx,
		`DELETE FROM menus WHERE id = $1`,
		id,
	)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return ErrInUse
		}

		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// =========================
// Top Menus Report
// =========================

func (s *PostgresStore) TopMenus(ctx context.Context, limit int) ([]TopMenu, error) {
	rows, err := s.Pool.Query(
		ctx,
		`SELECT m.name,
		        SUM(oi.qty) AS jan,
		        SUM(oi.qty * m.price)::bigint AS baht
		 FROM order_items oi
		 JOIN menus m ON m.id = oi.menu_id
		 GROUP BY m.name
		 ORDER BY baht DESC
		 LIMIT $1`,
		limit,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	result := []TopMenu{}

	for rows.Next() {
		var m TopMenu

		if err := rows.Scan(
			&m.Name,
			&m.Jan,
			&m.Baht,
		); err != nil {
			return nil, err
		}

		result = append(result, m)
	}

	return result, rows.Err()
}
