import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Clock } from "lucide-react";

import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import { initials } from "@/lib/marketing";
import { MonthYearPicker } from "@/components/compensation-policy/month-year-picker";
import {
  compensationPolicyKeys,
  getSalarySafety,
  type SalarySafetyEvaluation,
} from "@/services/compensation-policy";

function WorkHourCard({ evaluation }: { evaluation: SalarySafetyEvaluation }) {
  const target = evaluation.min_hours_per_month;
  const percent =
    target > 0
      ? Math.round((evaluation.monthly_active_hours / target) * 100)
      : 0;
  const barWidth = Math.min(100, Math.max(0, percent));

  return (
    <div className="space-y-4 rounded-lg border border-border bg-surface p-5">
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-surface-muted text-sm font-semibold text-text-primary">
          {initials(evaluation.full_name)}
        </div>
        <div>
          <p className="font-semibold text-text-primary">
            {evaluation.full_name}
          </p>
          <p className="text-sm text-text-secondary">
            {evaluation.monthly_active_hours.toFixed(1)} /{" "}
            {target.toFixed(0)} jam
          </p>
        </div>
      </div>

      <div className="space-y-1.5">
        <div className="flex items-center justify-between text-xs text-text-secondary">
          <span>Pencapaian target bulanan</span>
          <span className="font-semibold text-text-primary">{percent}%</span>
        </div>
        <div className="h-2 overflow-hidden rounded-full bg-surface-muted">
          <div
            className="h-full rounded-full bg-ops"
            style={{ width: `${barWidth}%` }}
          />
        </div>
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

export function TeamWorkHours() {
  const now = new Date();
  const [year, setYear] = useState(now.getFullYear());
  const [month, setMonth] = useState(now.getMonth() + 1);

  const query = useQuery({
    queryKey: compensationPolicyKeys.salarySafety(year, month),
    queryFn: () => getSalarySafety(year, month),
  });

  const evaluations = query.data ?? [];

  return (
    <Card className="space-y-6 p-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex items-start gap-3">
          <div className="rounded-md bg-ops-light p-3 text-ops">
            <Clock className="h-5 w-5" />
          </div>
          <div>
            <h2 className="font-display text-[18px] font-[700] text-text-primary">
              Jam Kerja Karyawan
            </h2>
            <p className="mt-1 text-sm text-text-secondary">
              Akumulasi jam aktif tiap karyawan pada periode terpilih.
            </p>
          </div>
        </div>
        <MonthYearPicker
          month={month}
          onMonthChange={setMonth}
          onYearChange={setYear}
          year={year}
        />
      </div>

      {query.isLoading ? (
        <p className="text-sm text-text-secondary">Memuat data...</p>
      ) : evaluations.length === 0 ? (
        <p className="text-sm text-text-secondary">
          Belum ada karyawan untuk dievaluasi.
        </p>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {evaluations.map((evaluation) => (
            <WorkHourCard evaluation={evaluation} key={evaluation.employee_id} />
          ))}
        </div>
      )}
    </Card>
  );
}
