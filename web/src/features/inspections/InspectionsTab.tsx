import { useState } from "react";
import { StatusDot } from "../../components/ui";
import { dateLabel } from "../../lib/format";
import { useInspections } from "./api";
import { LogInspectionDialog } from "./LogInspectionDialog";
import { WriteGuard } from "../../app/auth/WriteGuard";
import "./inspections.css";

export function InspectionsTab({ lineId }: { lineId: string }) {
  const { data: inspections = [], isLoading } = useInspections(lineId);
  const [open, setOpen] = useState(false);

  return (
    <>
      <div className="insp-bar">
        <h3 style={{ margin: 0 }}>Inspections</h3>
        <WriteGuard>
          <button className="btn" onClick={() => setOpen(true)}>Log inspection</button>
        </WriteGuard>
      </div>

      {isLoading ? (
        <p className="muted">Loading inspections…</p>
      ) : inspections.length === 0 ? (
        <div className="stub"><h3>No inspections yet</h3>Log the first inspection for this line.</div>
      ) : (
        <div className="insp-list">
          {inspections.map((i) => (
            <div className="insp-row" key={i.id}>
              <span className="insp-date">{dateLabel(i.inspected_at)}</span>
              <span className="insp-cond">
                <StatusDot condition={i.condition_status as never} /> {i.condition_status}
              </span>
              <span className="insp-meta">
                <span className="insp-by">{i.inspected_by || (i.source === "api" ? "Third-party API" : "—")}</span>
                {i.notes && <span className="insp-notes">{i.notes}</span>}
              </span>
            </div>
          ))}
        </div>
      )}

      {open && <LogInspectionDialog lineId={lineId} onClose={() => setOpen(false)} />}
    </>
  );
}
