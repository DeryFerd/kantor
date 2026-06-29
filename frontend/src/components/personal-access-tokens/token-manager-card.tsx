import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Copy, Info, KeyRound, Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { env } from "@/lib/env";
import {
  createPersonalAccessToken,
  listPersonalAccessTokens,
  personalAccessTokenKeys,
  revokePersonalAccessToken,
} from "@/services/personal-access-tokens";
import { toast } from "@/stores/toast-store";

function resolveMcpConfig() {
  const apiBase = env.VITE_API_BASE_URL;
  const origin = apiBase.startsWith("/")
    ? window.location.origin
    : apiBase.replace(/\/api\/v1\/?$/, "");
  return { mcpUrl: `${origin}/mcp`, baseUrl: origin, host: new URL(origin).host };
}

function formatTimestamp(value?: string | null) {
  if (!value) {
    return "-";
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleString("id-ID");
}

export function TokenManagerCard() {
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const [expiresInDays, setExpiresInDays] = useState("");
  const [createdToken, setCreatedToken] = useState<string | null>(null);
  const [isInfoOpen, setIsInfoOpen] = useState(false);
  const mcp = resolveMcpConfig();

  const tokensQuery = useQuery({
    queryKey: personalAccessTokenKeys.list(),
    queryFn: listPersonalAccessTokens,
  });

  const createMutation = useMutation({
    mutationFn: createPersonalAccessToken,
    onSuccess: async (result) => {
      setCreatedToken(result.token);
      setName("");
      setExpiresInDays("");
      toast.success("Access token berhasil dibuat");
      await queryClient.invalidateQueries({
        queryKey: personalAccessTokenKeys.list(),
      });
    },
    onError: (error) => {
      toast.error(
        "Gagal membuat access token",
        error instanceof Error ? error.message : undefined,
      );
    },
  });

  const revokeMutation = useMutation({
    mutationFn: revokePersonalAccessToken,
    onSuccess: async () => {
      toast.success("Access token dicabut");
      await queryClient.invalidateQueries({
        queryKey: personalAccessTokenKeys.list(),
      });
    },
    onError: (error) => {
      toast.error(
        "Gagal mencabut access token",
        error instanceof Error ? error.message : undefined,
      );
    },
  });

  const onCopy = async () => {
    if (!createdToken) {
      return;
    }
    await navigator.clipboard.writeText(createdToken);
    toast.success("Token disalin ke clipboard");
  };

  const onCreate = () => {
    const days = Number(expiresInDays);
    createMutation.mutate({
      name: name.trim(),
      expires_in_days: expiresInDays.trim() && days > 0 ? days : undefined,
    });
  };

  return (
    <Card className="space-y-5 p-6 xl:col-span-2">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          <div className="rounded-md bg-error-light p-3 text-error">
            <KeyRound className="h-5 w-5" />
          </div>
          <div>
            <h2 className="font-display text-[18px] font-[700] text-text-primary">
              Access Token (MCP)
            </h2>
            <p className="mt-1 text-sm text-text-secondary">
              Buat personal access token untuk binding KANTOR ke Claude / Hermes
              via MCP. Token mewarisi permission akun Anda.
            </p>
          </div>
        </div>
        <button
          aria-label="Cara hubungkan ke Claude Desktop"
          className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-border text-text-secondary transition hover:bg-surface-muted hover:text-text-primary"
          onClick={() => setIsInfoOpen(true)}
          type="button"
        >
          <Info className="h-4.5 w-4.5" />
        </button>
      </div>

      <Dialog onOpenChange={setIsInfoOpen} open={isInfoOpen}>
        <DialogContent size="lg">
          <DialogHeader>
            <DialogTitle>Hubungkan ke Claude Desktop</DialogTitle>
            <DialogDescription>
              Server MCP ini memakai Personal Access Token (Bearer), bukan OAuth.
            </DialogDescription>
          </DialogHeader>
          <DialogBody>
            <div className="space-y-4 text-sm text-text-secondary">
              <div>
                <p className="font-semibold text-text-primary">Endpoint MCP</p>
                <code className="mt-1 block overflow-x-auto rounded-md bg-surface-muted px-3 py-2 font-mono text-[13px] text-text-primary">
                  {mcp.mcpUrl}
                </code>
              </div>

              <div className="space-y-2">
                <p className="font-semibold text-text-primary">
                  Cara A — Connectors (remote, paling gampang)
                </p>
                <ol className="list-decimal space-y-1 pl-5">
                  <li>
                    Claude Desktop → Settings → Connectors → Add custom
                    connector.
                  </li>
                  <li>
                    Remote MCP server URL ={" "}
                    <code className="font-mono">{mcp.mcpUrl}</code> → Add.
                  </li>
                  <li>
                    Browser kebuka → login KANTOR → klik{" "}
                    <strong>Izinkan</strong>. OAuth Client ID/Secret biarkan
                    kosong.
                  </li>
                </ol>
              </div>

              <div className="space-y-2">
                <p className="font-semibold text-text-primary">
                  Cara B — Config file (stdio, pakai PAT)
                </p>
                <ol className="list-decimal space-y-1 pl-5">
                  <li>Buat token di kartu ini, lalu salin.</li>
                  <li>
                    Build binary:{" "}
                    <code className="font-mono">
                      go build -o kantor-mcp ./backend/cmd/mcp
                    </code>
                  </li>
                  <li>Claude Desktop → Settings → Developer → Edit Config.</li>
                  <li>
                    Tempel ke{" "}
                    <code className="font-mono">
                      claude_desktop_config.json
                    </code>
                    :
                  </li>
                </ol>
                <pre className="overflow-x-auto rounded-md bg-surface-muted px-3 py-3 font-mono text-[12px] leading-relaxed text-text-primary">
                  {`{
  "mcpServers": {
    "kantor": {
      "command": "/path/ke/kantor-mcp",
      "env": {
        "KANTOR_BASE_URL": "${mcp.baseUrl}",
        "KANTOR_PAT": "kantor_pat_...",
        "KANTOR_TENANT_HOST": "${mcp.host}"
      }
    }
  }
}`}
                </pre>
                <p>Restart Claude Desktop → tools KANTOR muncul.</p>
              </div>

              <div className="rounded-md border border-border bg-surface-muted/40 p-3 text-text-primary">
                <p className="font-semibold">OAuth Client ID / Secret?</p>
                <p className="mt-1 text-text-secondary">
                  Kosongkan saja — KANTOR pakai Dynamic Client Registration, jadi
                  Claude mendaftar sendiri otomatis.
                </p>
              </div>
            </div>
          </DialogBody>
        </DialogContent>
      </Dialog>

      {createdToken ? (
        <div className="space-y-2 rounded-md border border-warning/40 bg-warning-light p-4">
          <p className="text-sm font-semibold text-text-primary">
            Salin token sekarang. Token hanya ditampilkan satu kali.
          </p>
          <div className="flex items-center gap-2">
            <code className="flex-1 overflow-x-auto rounded-md bg-surface px-3 py-2 font-mono text-[13px] text-text-primary">
              {createdToken}
            </code>
            <Button onClick={onCopy} type="button" variant="secondary">
              <Copy className="h-4 w-4" />
              Salin
            </Button>
          </div>
        </div>
      ) : null}

      <div className="grid gap-4 md:grid-cols-[minmax(0,1fr)_180px_auto] md:items-end">
        <div className="space-y-1.5">
          <label
            className="text-[13px] font-[600] text-text-primary"
            htmlFor="pat-name"
          >
            Nama Token
          </label>
          <Input
            className="h-10 rounded-[6px] border-transparent bg-surface-muted px-3 text-[14px] focus:border-ops focus:bg-surface focus:ring-2 focus:ring-ops/20"
            id="pat-name"
            onChange={(event) => setName(event.target.value)}
            placeholder="claude-desktop"
            value={name}
          />
        </div>
        <div className="space-y-1.5">
          <label
            className="text-[13px] font-[600] text-text-primary"
            htmlFor="pat-expiry"
          >
            Kadaluarsa (hari)
          </label>
          <Input
            className="h-10 rounded-[6px] border-transparent bg-surface-muted px-3 text-[14px] focus:border-ops focus:bg-surface focus:ring-2 focus:ring-ops/20"
            id="pat-expiry"
            min={1}
            onChange={(event) => setExpiresInDays(event.target.value)}
            placeholder="opsional"
            type="number"
            value={expiresInDays}
          />
        </div>
        <Button
          disabled={createMutation.isPending || name.trim().length === 0}
          onClick={onCreate}
          type="button"
        >
          <Plus className="h-4 w-4" />
          Buat Token
        </Button>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-border text-xs uppercase tracking-[0.06em] text-text-secondary">
              <th className="py-2 pr-4 font-semibold">Nama</th>
              <th className="py-2 pr-4 font-semibold">Prefix</th>
              <th className="py-2 pr-4 font-semibold">Terakhir Dipakai</th>
              <th className="py-2 pr-4 font-semibold">Kadaluarsa</th>
              <th className="py-2 font-semibold" />
            </tr>
          </thead>
          <tbody>
            {tokensQuery.isLoading ? (
              <tr>
                <td className="py-4 text-text-secondary" colSpan={5}>
                  Memuat token...
                </td>
              </tr>
            ) : (tokensQuery.data ?? []).length === 0 ? (
              <tr>
                <td className="py-4 text-text-secondary" colSpan={5}>
                  Belum ada token.
                </td>
              </tr>
            ) : (
              (tokensQuery.data ?? []).map((token) => (
                <tr
                  className="border-b border-border/60 text-text-primary"
                  key={token.id}
                >
                  <td className="py-2 pr-4">{token.name}</td>
                  <td className="py-2 pr-4 font-mono text-[13px]">
                    {token.token_prefix}…
                  </td>
                  <td className="py-2 pr-4">
                    {formatTimestamp(token.last_used_at)}
                  </td>
                  <td className="py-2 pr-4">
                    {formatTimestamp(token.expires_at)}
                  </td>
                  <td className="py-2">
                    <button
                      className="inline-flex items-center gap-1 text-xs font-semibold text-error hover:underline"
                      disabled={revokeMutation.isPending}
                      onClick={() => revokeMutation.mutate(token.id)}
                      type="button"
                    >
                      <Trash2 className="h-4 w-4" />
                      Cabut
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </Card>
  );
}
