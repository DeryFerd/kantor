package middleware

import (
	"context"
	"net/http"
	"strings"

	backendauth "github.com/kana-consultant/kantor/backend/internal/auth"
	"github.com/kana-consultant/kantor/backend/internal/rbac"
	"github.com/kana-consultant/kantor/backend/internal/response"
)

type Principal struct {
	UserID       string
	TenantID     string
	IsSuperAdmin bool
	Roles        []string
	Permissions  []string
	ModuleRoles  map[string]rbac.ModuleRole
	Cached       *rbac.CachedPermissions
}

type contextKey string

const principalContextKey contextKey = "principal"

type patIdentity struct {
	userID   string
	tenantID string
}

func AuthMiddleware(
	parseToken func(string) (*backendauth.AccessClaims, error),
	loadPermissions func(context.Context, string) (*rbac.CachedPermissions, error),
	blacklist *backendauth.AccessTokenBlacklist,
	authenticatePAT func(context.Context, string) (string, string, error),
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := strings.TrimSpace(r.Header.Get("Authorization"))
			if header == "" {
				response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization header is required", nil)
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization header must use Bearer token", nil)
				return
			}

			rawToken := parts[1]

			var userID, tenantID string
			if backendauth.IsPersonalAccessToken(rawToken) {
				if authenticatePAT == nil {
					response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Access token is invalid or expired", nil)
					return
				}
				identity, err := WithScopedTenantConn(r.Context(), func(ctx context.Context) (patIdentity, error) {
					uid, tid, authErr := authenticatePAT(ctx, rawToken)
					return patIdentity{userID: uid, tenantID: tid}, authErr
				})
				if err != nil {
					response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Access token is invalid or expired", nil)
					return
				}
				userID, tenantID = identity.userID, identity.tenantID
			} else {
				claims, err := parseToken(rawToken)
				if err != nil {
					response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Access token is invalid or expired", nil)
					return
				}
				if blacklist != nil && blacklist.IsRevoked(claims.ID) {
					response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Access token has been revoked", nil)
					return
				}
				userID, tenantID = claims.Subject, claims.TenantID
			}

			cachedPermissions, err := WithScopedTenantConn(r.Context(), func(ctx context.Context) (*rbac.CachedPermissions, error) {
				return loadPermissions(ctx, userID)
			})
			if err != nil {
				response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Failed to resolve user permissions", nil)
				return
			}
			if !cachedPermissions.IsActive {
				response.WriteError(w, http.StatusForbidden, "INACTIVE_USER", "akun pengguna sedang tidak aktif", nil)
				return
			}

			roles := make([]string, 0, len(cachedPermissions.ModuleRoles)+1)
			if cachedPermissions.IsSuperAdmin {
				roles = append(roles, "super_admin")
			}
			for moduleID, role := range cachedPermissions.ModuleRoles {
				roles = append(roles, role.RoleSlug+":"+moduleID)
			}

			principal := Principal{
				UserID:       userID,
				TenantID:     tenantID,
				IsSuperAdmin: cachedPermissions.IsSuperAdmin,
				Roles:        roles,
				Permissions:  cachedPermissions.PermissionList(),
				ModuleRoles:  cachedPermissions.ModuleRoles,
				Cached:       cachedPermissions,
			}

			ctx := context.WithValue(r.Context(), principalContextKey, principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey).(Principal)
	return principal, ok
}
