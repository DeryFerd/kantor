import { authGetJSON, authRequestJSON } from "@/lib/api-client";

export type PersonalAccessToken = {
  id: string;
  name: string;
  token_prefix: string;
  last_used_at?: string | null;
  expires_at?: string | null;
  created_at: string;
};

export type CreatePersonalAccessTokenPayload = {
  name: string;
  expires_in_days?: number;
};

export type PersonalAccessTokenCreated = {
  token: string;
  token_info: PersonalAccessToken;
};

export const personalAccessTokenKeys = {
  all: ["personal-access-tokens"] as const,
  list: () => [...personalAccessTokenKeys.all, "list"] as const,
};

export function listPersonalAccessTokens(): Promise<PersonalAccessToken[]> {
  return authGetJSON<PersonalAccessToken[]>("/auth/pat");
}

export function createPersonalAccessToken(
  payload: CreatePersonalAccessTokenPayload,
): Promise<PersonalAccessTokenCreated> {
  return authRequestJSON<PersonalAccessTokenCreated>("/auth/pat", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export function revokePersonalAccessToken(
  tokenID: string,
): Promise<{ status: string }> {
  return authRequestJSON<{ status: string }>(`/auth/pat/${tokenID}`, {
    method: "DELETE",
  });
}
