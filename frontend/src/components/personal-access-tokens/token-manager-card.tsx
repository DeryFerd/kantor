import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Copy, KeyRound, Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  createPersonalAccessToken,
  listPersonalAccessTokens,
  personalAccessTokenKeys,
  revokePersonalAccessToken,
} from "@/services/personal-access-tokens";
import { toast } from "@/stores/toast-store";

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
