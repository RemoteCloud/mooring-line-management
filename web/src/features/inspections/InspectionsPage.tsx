import { API_BASE } from "../../config";
import { useVessel } from "../../app/VesselContext";
import { StatusDot } from "../../components/ui";
import { dateLabel } from "../../lib/format";
import { useLogbook } from "./api";
import "./inspections.css";

export function InspectionsPage() {
  const { vesselId } = useVessel();
  const { data: entries = [], isLoading } = useLogbook(vesselId);
  const recent = entries.slice(0, 20);

  return (
    <>
      <h1 className="page-title">Inspections</h1>
      <p className="page-sub">Recent inspections and downloadable condition reports.</p>

      {vesselId && (
        <div className="insp-downloads">
          <a href={`${API_BASE}/reports/condition?vessel_id=${vesselId}&format=csv`}>Download CSV</a>
          <a href={`${API_BASE}/reports/condition?vessel_id=${vesselId}&format=pdf`}>Download PDF</a>
        </div>
      )}

      {isLoading ? (
        <p className="muted">Loading inspections…</p>
      ) : recent.length === 0 ? (
        <div className="stub"><h3>No inspections yet</h3>Log inspections from a line's rope record.</div>
      ) : (
        <div className="insp-list">
          {recent.map((i) => (
            <div className="insp-row" key={i.id}>
              <span className="insp-date">{dateLabel(i.inspected_at)}</span>
              <span className="insp-cond">
                <StatusDot condition={i.condition_status as never} /> {i.condition_status}
              </span>
              <span className="insp-meta">
                <span className="insp-by">{i.line_name} · {i.serial_number}</span>
                {i.notes && <span className="insp-notes">{i.notes}</span>}
              </span>
            </div>
          ))}
        </div>
      )}
    </>
  );
}
