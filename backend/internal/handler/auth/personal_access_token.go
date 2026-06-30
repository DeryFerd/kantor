package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/kana-consultant/kantor/backend/internal/dto"
	"github.com/kana-consultant/kantor/backend/internal/httputil"
	platformmiddleware "github.com/kana-consultant/kantor/backend/internal/middleware"
	"github.com/kana-consultant/kantor/backend/internal/model"
	authrepo "github.com/kana-consultant/kantor/backend/internal/repository/auth"
	"github.com/kana-consultant/kantor/backend/internal/response"
	authservice "github.com/kana-consultant/kantor/backend/internal/service/auth"
)

type PATHandler struct {
	service   *authservice.PATService
	validator *validator.Validate
}

func NewPATHandler(service *authservice.PATService) *PATHandler {
	return &PATHandler{
		service:   service,
		validator: validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (h *PATHandler) RegisterRoutes(router chi.Router) {
	router.Post("/", h.create)
	router.Get("/", h.list)
	router.Delete("/{tokenID}", h.revoke)
}

func (h *PATHandler) create(w http.ResponseWriter, r *http.Request) {
	principal, ok := platformmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authenticated principal is missing", nil)
		return
	}

	var req dto.CreatePersonalAccessTokenRequest
	if !httputil.DecodeAndValidate(h.validator, w, r, &req) {
		return
	}

	token, item, err := h.service.Issue(r.Context(), principal.UserID, req.Name, req.ExpiresInDays, req.Scope, time.Now())
	if err != nil {
		platformmiddleware.LoggerFromContext(r.Context()).Error("create personal access token failed", "error", err, "user_id", principal.UserID)
		response.WriteInternalError(r.Context(), w, err, "Failed to create access token")
		return
	}

	response.WriteJSON(w, http.StatusCreated, dto.PersonalAccessTokenCreatedResponse{
		Token:     token,
		TokenInfo: toPATResponse(item),
	}, nil)
}

func (h *PATHandler) list(w http.ResponseWriter, r *http.Request) {
	principal, ok := platformmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authenticated principal is missing", nil)
		return
	}

	items, err := h.service.List(r.Context(), principal.UserID)
	if err != nil {
		platformmiddleware.LoggerFromContext(r.Context()).Error("list personal access tokens failed", "error", err, "user_id", principal.UserID)
		response.WriteInternalError(r.Context(), w, err, "Failed to load access tokens")
		return
	}

	out := make([]dto.PersonalAccessTokenResponse, 0, len(items))
	for _, item := range items {
		out = append(out, toPATResponse(item))
	}
	response.WriteJSON(w, http.StatusOK, out, nil)
}

func (h *PATHandler) revoke(w http.ResponseWriter, r *http.Request) {
	principal, ok := platformmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authenticated principal is missing", nil)
		return
	}

	tokenID := chi.URLParam(r, "tokenID")
	if err := h.service.Revoke(r.Context(), principal.UserID, tokenID); err != nil {
		if errors.Is(err, authrepo.ErrPersonalAccessTokenNotFound) {
			response.WriteError(w, http.StatusNotFound, "PERSONAL_ACCESS_TOKEN_NOT_FOUND", "Access token not found", nil)
			return
		}
		platformmiddleware.LoggerFromContext(r.Context()).Error("revoke personal access token failed", "error", err, "user_id", principal.UserID)
		response.WriteInternalError(r.Context(), w, err, "Failed to revoke access token")
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "revoked"}, nil)
}

func toPATResponse(item model.PersonalAccessToken) dto.PersonalAccessTokenResponse {
	res := dto.PersonalAccessTokenResponse{
		ID:          item.ID,
		Name:        item.Name,
		TokenPrefix: item.TokenPrefix,
		Scope:       item.Scope,
		CreatedAt:   item.CreatedAt.Format(time.RFC3339),
	}
	if item.LastUsedAt != nil {
		formatted := item.LastUsedAt.Format(time.RFC3339)
		res.LastUsedAt = &formatted
	}
	if item.ExpiresAt != nil {
		formatted := item.ExpiresAt.Format(time.RFC3339)
		res.ExpiresAt = &formatted
	}
	return res
}
