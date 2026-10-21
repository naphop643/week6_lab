package mw

import (
	"context"
	"net/http"
	"os"
	"strings"

	"moviehub/auth"
)

type contextKey string

const MemberClaimsKey contextKey = "memberClaims"

// RequireAuth ตรวจสอบ JWT ก่อนให้คำขอไปถึง handler
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		parts := strings.SplitN(header, " ", 2)

		if len(parts) != 2 ||
			!strings.EqualFold(parts[0], "Bearer") ||
			strings.TrimSpace(parts[1]) == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		claims, err := auth.ParseToken(strings.TrimSpace(parts[1]), secret)
		if err != nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), MemberClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin อนุญาตเฉพาะผู้ใช้ที่มี role เป็น admin
// ต้องใช้หลัง RequireAuth
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(MemberClaimsKey).(*auth.Claims)

		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		if claims.Role != "admin" {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
