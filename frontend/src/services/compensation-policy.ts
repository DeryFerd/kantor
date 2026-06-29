import { authGetJSON, authRequestJSON } from "@/lib/api-client";

export type CompensationPolicy = {
  monthly_base_salary: number;
  min_hours_per_day: number;
  min_hours_per_month: number;
  timezone: string;
  updated_at: string;
};

export type UpdateCompensationPolicyPayload = {
  monthly_base_salary: number;
  min_hours_per_day: number;
  min_hours_per_month: number;
  timezone: string;
};

export type SalarySafetyStatus = "safe" | "at_risk" | "no_data";

export type DailyHoursViolation = {
  date: string;
  active_hours: number;
};

export type SalarySafetyEvaluation = {
  employee_id: string;
  user_id?: string | null;
  full_name: string;
  period_year: number;
  period_month: number;
  monthly_active_hours: number;
  min_hours_per_month: number;
  min_hours_per_day: number;
  base_salary: number;
  daily_violations: DailyHoursViolation[];
  status: SalarySafetyStatus;
};

export const compensationPolicyKeys = {
  all: ["compensation-policy"] as const,
  policy: () => [...compensationPolicyKeys.all, "policy"] as const,
  salarySafety: (year: number, month: number) =>
    [...compensationPolicyKeys.all, "salary-safety", { year, month }] as const,
};

export function getCompensationPolicy(): Promise<CompensationPolicy> {
  return authGetJSON<CompensationPolicy>("/hris/compensation-policy");
}

export function updateCompensationPolicy(
  payload: UpdateCompensationPolicyPayload,
): Promise<CompensationPolicy> {
  return authRequestJSON<CompensationPolicy>("/hris/compensation-policy", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export function getSalarySafety(
  year: number,
  month: number,
): Promise<SalarySafetyEvaluation[]> {
  return authGetJSON<SalarySafetyEvaluation[]>(
    `/hris/salary-safety?year=${year}&month=${month}`,
  );
}

