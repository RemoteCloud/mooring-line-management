import { useVessel } from "../../app/VesselContext";
import { StatusDot } from "../../components/ui";
import { dateLabel } from "../../lib/format";
import { useLogbook } from "./api";
import "./inspections.css";

export function LogbookPage() {
  const { vesselId } = useVessel();
  const { data: entries = [], isLoading } = useLogbook(vesselId);

  return (
    <>
      <h1 className="page-title">Log book</h1>
      <p className="page-sub">Chronological inspection log across all lines.</p>

      {isLoading ? (
        <p className="muted">Loading log book…</p>
      ) : entries.length === 0 ? (
        <div className="stub"><h3>No inspections yet</h3>Inspections appear here as they are logged or ingested.</div>
      ) : (
        <div className="table-wrap">
          <table className="grid">
            <thead>
              <tr>
                <th>Date</th>
                <th>Line</th>
                <th>Serial</th>
                <th>Condition</th>
                <th>Inspector</th>
                <th>Notes</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((i) => (
                <tr key={i.id}>
                  <td>{dateLabel(i.inspected_at)}</td>
                  <td>{i.line_name}</td>
                  <td>{i.serial_number}</td>
                  <td><StatusDot condition={i.condition_status as never} /> {i.condition_status}</td>
                  <td>{i.inspected_by || (i.source === "api" ? "Third-party API" : "—")}</td>
                  <td>{i.notes || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}
