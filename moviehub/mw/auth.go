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
