import { useState } from "react";
import { useLogInspection } from "./api";

export function LogInspectionDialog({ lineId, onClose }: { lineId: string; onClose: () => void }) {
  const log = useLogInspection(lineId);

  const [form, setForm] = useState({
    conditionStatus: "Good" as "Good" | "Monitor" | "Action",
    inspectedBy: "",
    notes: "",
    inspectedAt: "",
  });
  const set = (k: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) =>
    setForm({ ...form, [k]: e.target.value });

  const submit = async () => {
    await log.mutateAsync({
      conditionStatus: form.conditionStatus,
      inspectedBy: form.inspectedBy || undefined,
      notes: form.notes || undefined,
      // datetime-local yields "YYYY-MM-DDTHH:mm"; append seconds + Z for RFC3339.
      inspectedAt: form.inspectedAt ? `${form.inspectedAt}:00Z` : undefined,
    });
    onClose();
  };

  return (
    <div className="overlay" onClick={onClose}>
      <div className="dialog" onClick={(e) => e.stopPropagation()}>
        <h3>Log inspection</h3>

        <div className="field">
          <label>Condition</label>
          <select className="input" value={form.conditionStatus} onChange={set("conditionStatus")}>
            <option value="Good">Good</option>
            <option value="Monitor">Monitor</option>
            <option value="Action">Action</option>
          </select>
        </div>

        <div className="field">
          <label>Inspected by</label>
          <input className="input" value={form.inspectedBy} onChange={set("inspectedBy")} />
        </div>

        <div className="field">
          <label>Date &amp; time (optional)</label>
          <input className="input" type="datetime-local" value={form.inspectedAt} onChange={set("inspectedAt")} />
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
