import { authRequestJSON } from "@/lib/api-client";

export type OAuthGrantPayload = {
  client_id: string;
  redirect_uri: string;
  code_challenge: string;
  code_challenge_method: string;
  scope?: string;
  state?: string;
};

export function approveOAuthGrant(
  payload: OAuthGrantPayload,
): Promise<{ redirect_uri: string }> {
  return authRequestJSON<{ redirect_uri: string }>("/oauth/grant", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}
