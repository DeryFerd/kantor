import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ShieldCheck } from "lucide-react";

import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import { EmployeeSafetyCard } from "@/components/compensation-policy/employee-safety-card";
import { MonthYearPicker } from "@/components/compensation-policy/month-year-picker";
import {
  compensationPolicyKeys,
  getSalarySafety,
} from "@/services/compensation-policy";

type SummaryStat = {
  label: string;
  value: number;
  accent: string;
};

export function SalarySafetyOverview() {
  const now = new Date();
  const [year, setYear] = useState(now.getFullYear());
  const [month, setMonth] = useState(now.getMonth() + 1);

  const safetyQuery = useQuery({
    queryKey: compensationPolicyKeys.salarySafety(year, month),
    queryFn: () => getSalarySafety(year, month),
  });

  const stats = useMemo<SummaryStat[]>(() => {
    const data = safetyQuery.data ?? [];
    const countBy = (status: string) =>
      data.filter((evaluation) => evaluation.status === status).length;
    return [
      { label: "Total Karyawan", value: data.length, accent: "text-text-primary" },
      { label: "Aman", value: countBy("safe"), accent: "text-success" },
      { label: "Berisiko", value: countBy("at_risk"), accent: "text-error" },
      { label: "Tanpa Data", value: countBy("no_data"), accent: "text-text-secondary" },
    ];
  }, [safetyQuery.data]);

  const evaluations = safetyQuery.data ?? [];

  return (
    <Card className="space-y-6 p-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex items-start gap-3">
          <div className="rounded-md bg-error-light p-3 text-error">
            <ShieldCheck className="h-5 w-5" />
          </div>
          <div>
            <h2 className="font-display text-[18px] font-[700] text-text-primary">
              Status Keamanan Gaji
            </h2>
            <p className="mt-1 text-sm text-text-secondary">
              Ringkasan kepatuhan jam kerja seluruh karyawan terhadap aturan
              kompensasi.
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

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        {stats.map((stat) => (
          <div
            className="rounded-lg border border-border bg-surface-muted/40 p-4"
            key={stat.label}
          >
            <p className="text-xs font-semibold uppercase tracking-[0.06em] text-text-secondary">
              {stat.label}
            </p>
            <p className={cn("mt-1 font-display text-[24px] font-[700]", stat.accent)}>
              {stat.value}
            </p>
          </div>
        ))}
      </div>

      {safetyQuery.isLoading ? (
        <p className="text-sm text-text-secondary">Memuat data...</p>
      ) : evaluations.length === 0 ? (
        <p className="text-sm text-text-secondary">
          Belum ada karyawan untuk dievaluasi.
        </p>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {evaluations.map((evaluation) => (
            <EmployeeSafetyCard
              evaluation={evaluation}
              key={evaluation.employee_id}
            />
          ))}
        </div>
      )}
    </Card>
  );
}
