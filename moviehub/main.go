package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"moviehub/handler"
	"moviehub/mw"
	"moviehub/store"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("กรุณากำหนด DATABASE_URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("เชื่อมต่อ PostgreSQL ไม่สำเร็จ: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ติดต่อ PostgreSQL ไม่สำเร็จ: %v", err)
	}

	log.Println("เชื่อมต่อ PostgreSQL สำเร็จ")

	st := &store.Store{Pool: pool}
	h := &handler.Handler{Store: st}

	mux := http.NewServeMux()

	// Authentication
	mux.HandleFunc("POST /api/register", h.Register)
	mux.HandleFunc("POST /api/login", h.Login)
	mux.Handle("GET /api/me",
		mw.RequireAuth(http.HandlerFunc(h.Me)),
	)

	// Movies
	mux.HandleFunc("GET /api/movies", h.SearchMovies)
	mux.HandleFunc("GET /api/movies/{id}/reviews", h.ListReviews)
	mux.Handle("POST /api/movies/{id}/reviews",
		mw.RequireAuth(http.HandlerFunc(h.CreateReview)),
	)

	// Admin: ต้อง Login และมี role เป็น admin
	mux.Handle("GET /api/admin/members",
		mw.RequireAuth(
			mw.RequireAdmin(http.HandlerFunc(h.ListMembers)),
		),
	)

	mux.Handle("GET /api/admin/stats",
		mw.RequireAuth(
			mw.RequireAdmin(http.HandlerFunc(h.AdminStats)),
		),
	)

	log.Println("MovieHub API พร้อมที่ http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mw.Logging(mux)))
}
