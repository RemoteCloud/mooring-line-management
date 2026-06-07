export function ageLabel(days: number | undefined): string {
  if (!days || days <= 0) return "—";
  if (days < 60) return `${days} d`;
  const months = Math.round(days / 30.44);
  if (months < 24) return `${months} mo`;
  return `${(days / 365.25).toFixed(1)} yr`;
}

export function dateLabel(iso: string | undefined | null): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" });
}

export type Condition = "Good" | "Monitor" | "Action" | "" | undefined;

export function condClass(c: Condition): string {
  switch (c) {
    case "Good":
      return "good";
    case "Monitor":
      return "monitor";
    case "Action":
      return "action";
    default:
      return "";
  }
}
