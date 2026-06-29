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

type MonthYearPickerProps = {
  year: number;
  month: number;
  onYearChange: (year: number) => void;
  onMonthChange: (month: number) => void;
};

export function MonthYearPicker({
  year,
  month,
  onYearChange,
  onMonthChange,
}: MonthYearPickerProps) {
  const currentYear = new Date().getFullYear();
  const yearOptions = [currentYear, currentYear - 1];

  return (
    <div className="flex gap-2">
      <select
        className="h-9 rounded-sm border-[1.5px] border-transparent bg-surface-muted px-2 text-[13px] text-text-primary outline-none focus:border-[#4C9AFF] focus:bg-surface"
        onChange={(event) => onMonthChange(Number(event.target.value))}
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
        onChange={(event) => onYearChange(Number(event.target.value))}
        value={year}
      >
        {yearOptions.map((option) => (
          <option key={option} value={option}>
            {option}
          </option>
        ))}
      </select>
    </div>
  );
}
