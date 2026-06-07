import { useState } from "react";
import { useLogInspection } from "./api";

export function LogInspectionDialog({ lineId, onClose }: { lineId: string; onClose: () => void }) {
  const log = useLogInspection(lineId);

  const [form, setForm] = useState({
    condition_status: "Good" as "Good" | "Monitor" | "Action",
    inspected_by: "",
    notes: "",
    inspected_at: "",
  });
  const set = (k: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) =>
    setForm({ ...form, [k]: e.target.value });

  const submit = async () => {
    await log.mutateAsync({
      condition_status: form.condition_status,
      inspected_by: form.inspected_by || undefined,
      notes: form.notes || undefined,
      // datetime-local yields "YYYY-MM-DDTHH:mm"; append seconds + Z for RFC3339.
      inspected_at: form.inspected_at ? `${form.inspected_at}:00Z` : undefined,
    });
    onClose();
  };

  return (
    <div className="overlay" onClick={onClose}>
      <div className="dialog" onClick={(e) => e.stopPropagation()}>
        <h3>Log inspection</h3>

        <div className="field">
          <label>Condition</label>
          <select className="input" value={form.condition_status} onChange={set("condition_status")}>
            <option value="Good">Good</option>
            <option value="Monitor">Monitor</option>
            <option value="Action">Action</option>
          </select>
        </div>

        <div className="field">
          <label>Inspected by</label>
          <input className="input" value={form.inspected_by} onChange={set("inspected_by")} />
        </div>

        <div className="field">
          <label>Date &amp; time (optional)</label>
          <input className="input" type="datetime-local" value={form.inspected_at} onChange={set("inspected_at")} />
        </div>

        <div className="field">
          <label>Notes</label>
          <textarea className="input" rows={3} value={form.notes} onChange={set("notes")} />
        </div>

        {log.isError && <div className="err">Could not log inspection.</div>}

        <div className="dialog-actions">
          <button className="btn ghost" onClick={onClose}>Cancel</button>
          <button className="btn" disabled={log.isPending} onClick={submit}>
            {log.isPending ? "Saving…" : "Log inspection"}
          </button>
        </div>
      </div>
    </div>
  );
}
