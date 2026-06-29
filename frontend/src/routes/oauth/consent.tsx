import { useMutation } from "@tanstack/react-query";
import { createFileRoute, redirect } from "@tanstack/react-router";
import { ShieldCheck } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { ensureAuthenticated } from "@/services/auth";
import { approveOAuthGrant } from "@/services/oauth";
import { useAuthStore } from "@/stores/auth-store";

type ConsentSearch = {
  client_id: string;
  redirect_uri: string;
  code_challenge: string;
  code_challenge_method: string;
  state: string;
  scope: string;
};

export const Route = createFileRoute("/oauth/consent")({
  validateSearch: (search: Record<string, unknown>): ConsentSearch => ({
    client_id: String(search.client_id ?? ""),
    redirect_uri: String(search.redirect_uri ?? ""),
    code_challenge: String(search.code_challenge ?? ""),
    code_challenge_method: String(search.code_challenge_method ?? "S256"),
    state: String(search.state ?? ""),
    scope: String(search.scope ?? ""),
  }),
  beforeLoad: async () => {
    const session = await ensureAuthenticated();
    if (!session) {
      throw redirect({
        to: "/login",
        search: { redirect: window.location.pathname + window.location.search },
      });
    }
  },
  component: ConsentPage,
});

function ConsentPage() {
  const search = Route.useSearch();
  const session = useAuthStore((state) => state.session);

  const approveMutation = useMutation({
    mutationFn: approveOAuthGrant,
    onSuccess: (result) => {
      window.location.href = result.redirect_uri;
    },
  });

  const onApprove = () => {
    approveMutation.mutate({
      client_id: search.client_id,
      redirect_uri: search.redirect_uri,
      code_challenge: search.code_challenge,
      code_challenge_method: search.code_challenge_method,
      scope: search.scope || undefined,
      state: search.state || undefined,
    });
  };

  const onDeny = () => {
    const url = new URL(search.redirect_uri);
    url.searchParams.set("error", "access_denied");
    if (search.state) {
      url.searchParams.set("state", search.state);
    }
    window.location.href = url.toString();
  };

  const incomplete =
    !search.client_id || !search.redirect_uri || !search.code_challenge;

  return (
    <div className="flex min-h-dvh items-center justify-center bg-surface-muted p-4">
      <Card className="w-full max-w-md space-y-6 p-8">
        <div className="flex items-start gap-3">
          <div className="rounded-md bg-error-light p-3 text-error">
            <ShieldCheck className="h-6 w-6" />
          </div>
          <div>
            <h1 className="font-display text-[22px] font-[700] text-text-primary">
              Izinkan akses MCP?
            </h1>
            <p className="mt-1 text-sm leading-6 text-text-secondary">
              Aplikasi meminta akses ke KANTOR sebagai{" "}
              <strong className="text-text-primary">
                {session?.user.email}
              </strong>{" "}
              lewat MCP. Token mewarisi permission akun Anda.
            </p>
          </div>
        </div>

        {approveMutation.isError ? (
          <p className="text-sm text-error">
            Gagal memproses persetujuan. Coba lagi.
          </p>
        ) : null}

        {incomplete ? (
          <p className="text-sm text-error">
            Permintaan OAuth tidak lengkap atau tidak valid.
          </p>
        ) : (
          <div className="flex justify-end gap-3">
            <Button onClick={onDeny} type="button" variant="outline">
              Tolak
            </Button>
            <Button
              disabled={approveMutation.isPending}
              onClick={onApprove}
              type="button"
            >
              Izinkan
            </Button>
          </div>
        )}
      </Card>
    </div>
  );
}
