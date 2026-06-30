package dto

type CreatePersonalAccessTokenRequest struct {
	Name          string  `json:"name" validate:"required,min=1,max=120"`
	ExpiresInDays *int    `json:"expires_in_days,omitempty" validate:"omitempty,min=1,max=365"`
	Scope         *string `json:"scope,omitempty" validate:"omitempty,oneof=tracker"`
}

type PersonalAccessTokenResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	TokenPrefix string  `json:"token_prefix"`
	Scope       *string `json:"scope,omitempty"`
	LastUsedAt  *string `json:"last_used_at,omitempty"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

type PersonalAccessTokenCreatedResponse struct {
	Token     string                      `json:"token"`
	TokenInfo PersonalAccessTokenResponse `json:"token_info"`
}
