package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"moviehub/auth"
	"moviehub/mw"
	"moviehub/store"
)

type Handler struct {
	Store *store.Store
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("เขียน JSON ไม่สำเร็จ: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

func internal(w http.ResponseWriter, err error) {
	log.Printf("internal server error: %v", err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}

func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func (h *Handler) SearchMovies(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")

	movies, err := h.Store.SearchMovies(r.Context(), q)
	if err != nil {
		internal(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": movies,
	})
}

func (h *Handler) ListReviews(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid movie id")
		return
	}

	reviews, err := h.Store.ListReviews(r.Context(), id, 0)
	if err != nil {
		internal(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": reviews,
	})
}

// Register สมัครสมาชิกใหม่
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"displayName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	input.Email = strings.TrimSpace(input.Email)
	input.DisplayName = strings.TrimSpace(input.DisplayName)

	if input.Email == "" ||
		input.DisplayName == "" ||
		len(input.Password) < 8 ||
		len(input.Password) > 72 {
		writeError(w, http.StatusBadRequest, "invalid registration data")
		return
	}

	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid password")
		return
	}

	member, err := h.Store.CreateMember(
		r.Context(),
		input.Email,
		hash,
		input.DisplayName,
	)
	if err != nil {
		if errors.Is(err, store.ErrEmailTaken) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}

		internal(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, member)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	input.Email = strings.TrimSpace(input.Email)
	if input.Email == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	member, err := h.Store.GetMemberByEmail(r.Context(), input.Email)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		internal(w, err)
		return
	}

	if !auth.CheckPassword(member.PasswordHash, input.Password) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		internal(w, errors.New("JWT_SECRET is not configured"))
		return
	}

	token, err := auth.GenerateToken(
		member.ID,
		member.Email,
		member.Role,
		secret,
	)
	if err != nil {
		internal(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"member": map[string]any{
			"id":          member.ID,
			"email":       member.Email,
			"displayName": member.DisplayName,
			"role":        member.Role,
		},
	})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(mw.MemberClaimsKey).(*auth.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":    claims.MemberID,
		"email": claims.Email,
		"role":  claims.Role,
	})
}

func (h *Handler) CreateReview(w http.ResponseWriter, r *http.Request) {
	movieID, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid movie id")
		return
	}

	claims, ok := r.Context().Value(mw.MemberClaimsKey).(*auth.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input struct {
		Text string `json:"text"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	input.Text = strings.TrimSpace(input.Text)
	if len([]rune(input.Text)) < 10 {
		writeError(w, http.StatusBadRequest, "review must be at least 10 characters")
		return
	}

	review, err := h.Store.CreateReview(
		r.Context(),
		movieID,
		claims.MemberID,
		input.Text,
	)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "movie not found")
		return
	}

	if err != nil {
		internal(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, review)
}
