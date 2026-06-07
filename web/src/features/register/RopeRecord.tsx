import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useLine, type Line } from "../../api/hooks";
import { StatusDot, CopyButton, LifecycleBadge } from "../../components/ui";
import { ageLabel, dateLabel } from "../../lib/format";

const TABS = ["Overview", "Side tracking", "Inspections", "Files & photos"] as const;
type Tab = (typeof TABS)[number];

export function RopeRecord() {
  const { id } = useParams();
  const { data: l, isLoading } = useLine(id);
  const [tab, setTab] = useState<Tab>("Overview");

  if (isLoading || !l) return <p className="muted">Loading rope record…</p>;

  return (
    <>
      <Link to="/register" className="muted">← Register</Link>
      <div className="record-head" style={{ marginTop: 10 }}>
        <div>
          <h1 className="page-title" style={{ margin: 0 }}>
            <StatusDot condition={l.current_condition_status as never} /> {l.name}
          </h1>
          <div className="muted">{l.product_name} · {l.maker_name} · {l.line_type_name}</div>
        </div>
        <div className="grow" style={{ flex: 1 }} />
        <div className="record-meta">
          <div><b>{l.current_side || "n/a"}</b>side in use</div>
          <div><b>{ageLabel(l.install_age_days)}</b>install age</div>
          <div><b>{ageLabel(l.build_age_days)}</b>build age</div>
          <div><b>{dateLabel(l.next_inspection_due)}</b>next inspection</div>
          <div><b><LifecycleBadge status={l.lifecycle_status} /></b>lifecycle</div>
        </div>
      </div>

      <div className="record-meta" style={{ marginBottom: 16 }}>
        <div>Tag #&nbsp;<CopyButton value={l.tag_number ?? ""} /></div>
        <div>Certificate #&nbsp;<CopyButton value={l.certificate_number ?? ""} /></div>
        <div>Serial&nbsp;<CopyButton value={l.serial_number} /></div>
      </div>

      <div className="tabs">
        {TABS.map((t) => (
          <button key={t} className={"tab" + (tab === t ? " active" : "")} onClick={() => setTab(t)}>{t}</button>
        ))}
      </div>

      {tab === "Overview" && <Overview l={l} />}
      {tab === "Side tracking" && <SideTracking l={l} />}
      {tab === "Inspections" && <Empty what="inspections" note="Inspections arrive via the third-party API and manual logging — coming with the inspections slice." />}
      {tab === "Files & photos" && <Empty what="files" note="Condition photos, certificates and manuals — coming with the files slice." />}
    </>
  );
}

function Overview({ l }: { l: Line }) {
  return (
    <>
      <dl className="kv">
        <dt>Product</dt><dd>{l.product_name}</dd>
        <dt>Maker</dt><dd>{l.maker_name}</dd>
        <dt>Line type</dt><dd>{l.line_type_name}</dd>
        <dt>Construction</dt><dd>{l.construction_type || "—"}</dd>
        <dt>Length</dt><dd>{l.length ? `${l.length} m` : "—"}</dd>
        <dt>Serial</dt><dd>{l.serial_number}</dd>
        <dt>Location</dt><dd>{l.location_label}</dd>
        <dt>Manufactured</dt><dd>{dateLabel(l.manufacture_date)}</dd>
        <dt>Installed</dt><dd>{dateLabel(l.installation_date)}</dd>
        <dt>Turnable</dt><dd>{l.can_be_turned ? "Yes" : "No"}</dd>
      </dl>

      {l.components.length > 0 && (
        <>
          <h3 style={{ marginTop: 24 }}>Components</h3>
          <div className="table-wrap" style={{ maxWidth: 640 }}>
            <table className="grid">
              <thead><tr><th>Component</th><th>Type</th><th>Serial</th><th>Certificate</th></tr></thead>
              <tbody>
                {l.components.map((c) => (
                  <tr key={c.id}>
                    <td>{c.name}</td><td>{c.line_type_name}</td><td>{c.serial_number}</td><td>{c.certificate_number || "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </>
  );
}

function SideTracking({ l }: { l: Line }) {
  if (l.current_side === "n/a" || !l.can_be_turned) {
    return <p className="muted">This line is not reversible — no side tracking.</p>;
  }
  const card = (side: "A" | "B", age: number, change: string | null | undefined, cond: string | undefined) => (
    <div className={"side-card" + (l.current_side === side ? " active" : "")}>
      <h4>
        Side {side}
        {l.current_side === side && <span className="tag-active">● in use</span>}
      </h4>
      <dl className="kv" style={{ gridTemplateColumns: "130px 1fr" }}>
        <dt>Accumulated age</dt><dd>{ageLabel(age)}</dd>
        <dt>Last change</dt><dd>{dateLabel(change)}</dd>
        <dt>Condition</dt><dd><StatusDot condition={cond as never} /> {cond || "—"}</dd>
      </dl>
    </div>
  );
  return (
    <>
      {l.turn_due && <div className="card" style={{ borderColor: "var(--monitor)", marginBottom: 16, maxWidth: 640 }}>
        <b>Turn recommended</b> — the active side has reached the 6-month turn interval.
      </div>}
      <div className="side-cards">
        {card("A", l.side_a_age_days, l.side_a_change_date, l.side_a_condition)}
        {card("B", l.side_b_age_days, l.side_b_change_date, l.side_b_condition)}
      </div>
      <p className="muted" style={{ marginTop: 14 }}>The Turn action arrives with the turning slice.</p>
    </>
  );
}

function Empty({ what, note }: { what: string; note: string }) {
  return <div className="stub"><h3>No {what} yet</h3>{note}</div>;
}
