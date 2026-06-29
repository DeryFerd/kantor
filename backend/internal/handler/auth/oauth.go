package auth

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/kana-consultant/kantor/backend/internal/dto"
	platformmiddleware "github.com/kana-consultant/kantor/backend/internal/middleware"
	"github.com/kana-consultant/kantor/backend/internal/response"
	authservice "github.com/kana-consultant/kantor/backend/internal/service/auth"
)

type OAuthHandler struct {
	service *authservice.OAuthService
}

func NewOAuthHandler(service *authservice.OAuthService) *OAuthHandler {
	return &OAuthHandler{service: service}
}

func (h *OAuthHandler) AuthorizationServerMetadata(w http.ResponseWriter, r *http.Request) {
	base := requestPublicBaseURL(r, false)
	writeRawJSON(w, http.StatusOK, map[string]interface{}{
		"issuer":                                base,
		"authorization_endpoint":                base + "/oauth/authorize",
		"token_endpoint":                        base + "/oauth/token",
		"registration_endpoint":                 base + "/oauth/register",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"scopes_supported":                      []string{"mcp"},
	})
}

func (h *OAuthHandler) ProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	base := requestPublicBaseURL(r, false)
	writeRawJSON(w, http.StatusOK, map[string]interface{}{
		"resource":              base + "/mcp",
		"authorization_servers": []string{base},
	})
}

func (h *OAuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClientName   string   `json:"client_name"`
		RedirectURIs []string `json:"redirect_uris"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "invalid request body")
		return
	}

	result, err := h.service.RegisterClient(r.Context(), req.ClientName, req.RedirectURIs)
	if err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", err.Error())
		return
	}

	writeRawJSON(w, http.StatusCreated, map[string]interface{}{
		"client_id":                  result.ClientID,
		"client_name":                result.ClientName,
		"redirect_uris":              result.RedirectURIs,
		"token_endpoint_auth_method": "none",
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
	})
}

func (h *OAuthHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if query.Get("response_type") != "code" {
		http.Error(w, "unsupported_response_type", http.StatusBadRequest)
		return
	}

	clientID := query.Get("client_id")
	redirectURI := query.Get("redirect_uri")
	if err := h.service.ValidateAuthorize(r.Context(), clientID, redirectURI); err != nil {
		http.Error(w, "invalid client_id or redirect_uri", http.StatusBadRequest)
		return
	}

	consent := url.Values{}
	consent.Set("client_id", clientID)
	consent.Set("redirect_uri", redirectURI)
	consent.Set("code_challenge", query.Get("code_challenge"))
	consent.Set("code_challenge_method", query.Get("code_challenge_method"))
	consent.Set("state", query.Get("state"))
	consent.Set("scope", query.Get("scope"))
	http.Redirect(w, r, requestPublicBaseURL(r, false)+"/oauth/consent?"+consent.Encode(), http.StatusFound)
}

func (h *OAuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "could not parse request")
		return
	}

	switch r.PostFormValue("grant_type") {
	case "authorization_code":
		tokens, err := h.service.ExchangeCode(
			r.Context(),
			r.PostFormValue("code"),
			r.PostFormValue("client_id"),
			r.PostFormValue("redirect_uri"),
			r.PostFormValue("code_verifier"),
			r.UserAgent(),
			clientIP(r),
			time.Now(),
		)
		if err != nil {
			writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "authorization code is invalid or expired")
			return
		}
		writeTokenResponse(w, tokens)
	case "refresh_token":
		tokens, err := h.service.RefreshGrant(r.Context(), r.PostFormValue("refresh_token"), r.UserAgent(), clientIP(r))
		if err != nil {
			writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "refresh token is invalid or expired")
			return
		}
		writeTokenResponse(w, tokens)
	default:
		writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type", "grant_type not supported")
	}
}

func (h *OAuthHandler) Grant(w http.ResponseWriter, r *http.Request) {
	principal, ok := platformmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authenticated principal is missing", nil)
		return
	}

	var req struct {
		ClientID            string `json:"client_id"`
		RedirectURI         string `json:"redirect_uri"`
		CodeChallenge       string `json:"code_challenge"`
		CodeChallengeMethod string `json:"code_challenge_method"`
		Scope               string `json:"scope"`
		State               string `json:"state"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	code, err := h.service.Grant(r.Context(), authservice.GrantParams{
		UserID:              principal.UserID,
		ClientID:            req.ClientID,
		RedirectURI:         req.RedirectURI,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		Scope:               req.Scope,
	}, time.Now())
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "OAUTH_INVALID_REQUEST", "Permintaan OAuth tidak valid", nil)
		return
	}

	redirect, err := url.Parse(req.RedirectURI)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "OAUTH_INVALID_REQUEST", "Redirect URI tidak valid", nil)
		return
	}
	query := redirect.Query()
	query.Set("code", code)
	if req.State != "" {
		query.Set("state", req.State)
	}
	redirect.RawQuery = query.Encode()

	response.WriteJSON(w, http.StatusOK, map[string]string{"redirect_uri": redirect.String()}, nil)
}

func writeRawJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeOAuthError(w http.ResponseWriter, status int, code string, description string) {
	writeRawJSON(w, status, map[string]string{"error": code, "error_description": description})
}

func writeTokenResponse(w http.ResponseWriter, tokens dto.TokenPair) {
	writeRawJSON(w, http.StatusOK, map[string]interface{}{
		"access_token":  tokens.AccessToken,
		"token_type":    "Bearer",
		"expires_in":    tokens.ExpiresIn,
		"refresh_token": tokens.RefreshToken,
		"scope":         "mcp",
	})
}
