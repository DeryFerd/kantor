import { cn } from "@/lib/utils";
import { formatIDR } from "@/lib/currency";
import { initials } from "@/lib/marketing";
import {
  type SalarySafetyEvaluation,
  type SalarySafetyStatus,
} from "@/services/compensation-policy";

const statusBadge: Record<SalarySafetyStatus, string> = {
  safe: "bg-success-light text-success",
  at_risk: "bg-error-light text-error",
  no_data: "bg-surface-muted text-text-secondary",
};

const statusBar: Record<SalarySafetyStatus, string> = {
  safe: "bg-success",
  at_risk: "bg-error",
  no_data: "bg-border",
};

const statusLabels: Record<SalarySafetyStatus, string> = {
  safe: "Aman",
  at_risk: "Berisiko",
  no_data: "Tidak ada data",
};

type EmployeeSafetyCardProps = {
  evaluation: SalarySafetyEvaluation;
};

export function EmployeeSafetyCard({ evaluation }: EmployeeSafetyCardProps) {
  const target = evaluation.min_hours_per_month;
  const percent =
    target > 0
      ? Math.round((evaluation.monthly_active_hours / target) * 100)
      : 0;
  const barWidth = Math.min(100, Math.max(0, percent));

  return (
    <div className="space-y-4 rounded-lg border border-border bg-surface p-5">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-surface-muted text-sm font-semibold text-text-primary">
            {initials(evaluation.full_name)}
          </div>
          <div>
            <p className="font-semibold text-text-primary">
              {evaluation.full_name}
            </p>
            <p className="text-sm text-text-secondary">
              {formatIDR(evaluation.base_salary)}
            </p>
          </div>
        </div>
        <span
          className={cn(
            "inline-flex rounded-full px-2.5 py-1 text-[11px] font-semibold uppercase tracking-[0.06em]",
            statusBadge[evaluation.status],
          )}
        >
          {statusLabels[evaluation.status]}
        </span>
      </div>

      <div className="space-y-1.5">
        <div className="flex items-center justify-between text-xs text-text-secondary">
          <span>Jam bulan ini</span>
          <span className="font-semibold text-text-primary">
            {evaluation.monthly_active_hours.toFixed(1)} / {target.toFixed(0)}{" "}
            jam
          </span>
        </div>
        <div className="h-2 overflow-hidden rounded-full bg-surface-muted">
          <div
            className={cn("h-full rounded-full", statusBar[evaluation.status])}
            style={{ width: `${barWidth}%` }}
          />
        </div>
        <p className="text-right text-[11px] text-text-secondary">
          {percent}% dari target bulanan
        </p>
      </div>

      <div className="flex items-center justify-between rounded-md bg-surface-muted/60 px-3 py-2 text-sm">
        <span className="text-text-secondary">
          Hari di bawah {evaluation.min_hours_per_day.toFixed(0)} jam
        </span>
        <span
          className={cn(
            "font-semibold",
            evaluation.daily_violations.length > 0
              ? "text-error"
              : "text-text-primary",
          )}
        >
          {evaluation.daily_violations.length}
        </span>
      </div>
    </div>
  );
}
