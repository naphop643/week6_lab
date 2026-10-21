package store

import "context"

type AdminStats struct {
	Movies   int64 `json:"movies"`
	Members  int64 `json:"members"`
	Admins   int64 `json:"admins"`
	Votes    int64 `json:"votes"`
	Wishlist int64 `json:"wishlistItems"`
	Reviews  int64 `json:"reviews"`
}

func (s *Store) GetAdminStats(ctx context.Context) (AdminStats, error) {
	var stats AdminStats

	err := s.Pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM movies),
			(SELECT COUNT(*) FROM members),
			(SELECT COUNT(*) FROM members WHERE role = 'admin'),
			(SELECT COUNT(*) FROM votes),
			(SELECT COUNT(*) FROM wishlist_items),
			(SELECT COUNT(*) FROM reviews)
	`).Scan(
		&stats.Movies,
		&stats.Members,
		&stats.Admins,
		&stats.Votes,
		&stats.Wishlist,
		&stats.Reviews,
	)

	if err != nil {
		return AdminStats{}, err
	}

	return stats, nil
}
