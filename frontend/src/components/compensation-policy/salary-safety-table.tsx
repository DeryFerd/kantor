import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ShieldCheck } from "lucide-react";

import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import {
  compensationPolicyKeys,
  getSalarySafety,
  type SalarySafetyStatus,
} from "@/services/compensation-policy";

const monthLabels = [
  "Januari",
  "Februari",
  "Maret",
  "April",
  "Mei",
  "Juni",
  "Juli",
  "Agustus",
  "September",
  "Oktober",
  "November",
  "Desember",
];

const statusStyles: Record<SalarySafetyStatus, string> = {
  safe: "bg-success-light text-success",
  at_risk: "bg-error-light text-error",
  no_data: "bg-surface-muted text-text-secondary",
};

const statusLabels: Record<SalarySafetyStatus, string> = {
  safe: "Aman",
  at_risk: "Berisiko",
  no_data: "Tidak ada data",
};

const rupiahFormatter = new Intl.NumberFormat("id-ID", {
  style: "currency",
  currency: "IDR",
  maximumFractionDigits: 0,
});

export function SalarySafetyTable() {
  const now = new Date();
  const [year, setYear] = useState(now.getFullYear());
  const [month, setMonth] = useState(now.getMonth() + 1);

  const safetyQuery = useQuery({
    queryKey: compensationPolicyKeys.salarySafety(year, month),
    queryFn: () => getSalarySafety(year, month),
  });

  const yearOptions = [now.getFullYear(), now.getFullYear() - 1];

  return (
    <Card className="space-y-5 p-6 xl:col-span-2">
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-start gap-3">
          <div className="rounded-md bg-error-light p-3 text-error">
            <ShieldCheck className="h-5 w-5" />
          </div>
          <div>
            <h2 className="font-display text-[18px] font-[700] text-text-primary">
              Status Keamanan Gaji
            </h2>
            <p className="mt-1 text-sm text-text-secondary">
              Evaluasi jam kerja karyawan terhadap aturan kompensasi pada periode
              terpilih.
            </p>
          </div>
        </div>

        <div className="flex gap-2">
          <select
            className="h-9 rounded-sm border-[1.5px] border-transparent bg-surface-muted px-2 text-[13px] text-text-primary outline-none focus:border-[#4C9AFF] focus:bg-surface"
            onChange={(event) => setMonth(Number(event.target.value))}
            value={month}
          >
            {monthLabels.map((label, index) => (
              <option key={label} value={index + 1}>
                {label}
              </option>
            ))}
          </select>
          <select
            className="h-9 rounded-sm border-[1.5px] border-transparent bg-surface-muted px-2 text-[13px] text-text-primary outline-none focus:border-[#4C9AFF] focus:bg-surface"
            onChange={(event) => setYear(Number(event.target.value))}
            value={year}
          >
            {yearOptions.map((option) => (
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-border text-xs uppercase tracking-[0.06em] text-text-secondary">
              <th className="py-2 pr-4 font-semibold">Karyawan</th>
              <th className="py-2 pr-4 font-semibold">Gaji</th>
              <th className="py-2 pr-4 font-semibold">Jam Bulan Ini</th>
              <th className="py-2 pr-4 font-semibold">Min Bulan</th>
              <th className="py-2 pr-4 font-semibold">Pelanggaran Harian</th>
              <th className="py-2 font-semibold">Status</th>
            </tr>
          </thead>
          <tbody>
            {safetyQuery.isLoading ? (
              <tr>
                <td className="py-4 text-text-secondary" colSpan={6}>
                  Memuat data...
                </td>
              </tr>
            ) : (safetyQuery.data ?? []).length === 0 ? (
              <tr>
                <td className="py-4 text-text-secondary" colSpan={6}>
                  Belum ada karyawan untuk dievaluasi.
                </td>
              </tr>
            ) : (
              (safetyQuery.data ?? []).map((evaluation) => (
                <tr
                  className="border-b border-border/60 text-text-primary"
                  key={evaluation.employee_id}
                >
                  <td className="py-2 pr-4">{evaluation.full_name}</td>
                  <td className="py-2 pr-4">
                    {rupiahFormatter.format(evaluation.base_salary)}
                  </td>
                  <td className="py-2 pr-4">
                    {evaluation.monthly_active_hours.toFixed(2)} jam
                  </td>
                  <td className="py-2 pr-4">
                    {evaluation.min_hours_per_month.toFixed(0)} jam
                  </td>
                  <td className="py-2 pr-4">
                    {evaluation.daily_violations.length}
                  </td>
                  <td className="py-2">
                    <span
                      className={cn(
                        "inline-flex rounded-full px-2 py-0.5 text-[11px] font-semibold uppercase tracking-[0.06em]",
                        statusStyles[evaluation.status],
                      )}
                    >
                      {statusLabels[evaluation.status]}
                    </span>
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
