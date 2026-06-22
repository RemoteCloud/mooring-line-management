import { API_BASE } from "../../config";
import { getApiKey } from "../../api/authKey";
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
          {/* Browser navigations can't carry an Authorization header, so the key rides
              the query string (the API accepts ?api_key= as a fallback). */}
          <a href={`${API_BASE}/reports/condition?vesselId=${vesselId}&format=csv&api_key=${encodeURIComponent(getApiKey())}`}>Download CSV</a>
          <a href={`${API_BASE}/reports/condition?vesselId=${vesselId}&format=pdf&api_key=${encodeURIComponent(getApiKey())}`}>Download PDF</a>
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
              <span className="insp-date">{dateLabel(i.inspectedAt)}</span>
              <span className="insp-cond">
                <StatusDot condition={i.conditionStatus as never} /> {i.conditionStatus}
              </span>
              <span className="insp-meta">
                <span className="insp-by">{i.lineName} · {i.serialNumber}</span>
                {i.notes && <span className="insp-notes">{i.notes}</span>}
              </span>
            </div>
          ))}
        </div>
      )}
    </>
  );
}
