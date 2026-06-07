import { useState } from "react";
import { condClass, type Condition } from "../lib/format";

export function StatusDot({ condition }: { condition: Condition }) {
  const c = condClass(condition);
  return <span className={"dot " + (c || "")} title={condition || "no condition"} style={c ? undefined : { background: "var(--border)" }} />;
}

export function CopyButton({ value, label }: { value: string; label?: string }) {
  const [copied, setCopied] = useState(false);
  if (!value) return <span className="muted">—</span>;
  return (
    <button
      className="copy-btn"
      onClick={() => {
        navigator.clipboard?.writeText(value);
        setCopied(true);
        setTimeout(() => setCopied(false), 1200);
      }}
      title="Copy"
    >
      {label ?? value} <span className="copy-ico">{copied ? "✓" : "⧉"}</span>
    </button>
  );
}

export function LifecycleBadge({ status }: { status: string }) {
  const map: Record<string, string> = {
    ordered: "badge-ordered",
    active: "badge-active",
    spare: "badge-spare",
    retired: "badge-retired",
  };
  return <span className={"badge " + (map[status] || "")}>{status}</span>;
}
