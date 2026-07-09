import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useLine, type Line } from "../../api/hooks";
import { StatusDot, CopyButton, LifecycleBadge } from "../../components/ui";
import { ageLabel, dateLabel, condClass } from "../../lib/format";
import { TurnButton } from "../turning/TurnButton";
import { InspectionsTab } from "../inspections/InspectionsTab";
import { FilesTab } from "../files/FilesTab";

const TABS = ["Overview", "Side tracking", "Inspections", "Files & photos"] as const;
type Tab = (typeof TABS)[number];

export function RopeRecord() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { data: l, isLoading, isError, error } = useLine(id);
  const [tab, setTab] = useState<Tab>("Overview");

  if (isLoading) return <p className="muted">Loading rope record…</p>;

  if (isError || !l) {
    const status = (error as { status?: number } | null)?.status;
    return (
      <div className="empty-state">
        <Link to="/register" className="muted">← Register</Link>
        <h1 className="page-title" style={{ marginTop: 10 }}>Rope record</h1>
        <p className="muted">
          {status === 404
            ? "This rope record was not found — it may have been removed."
            : "Failed to load this rope record. Check the connection and try again."}
        </p>
        <Link to="/register" className="btn">Back to register</Link>
      </div>
    );
  }

  const onDeck = Boolean(l.currentDrumId || l.currentStorageId);

  return (
    <>
      <div className="record-bar">
        <Link to="/register" className="muted">← Register</Link>
        {onDeck && (
          <button type="button" className="btn" onClick={() => navigate(`/deck?line=${l.id}`)}>
            ⚓ Show on deck map
          </button>
        )}
      </div>
      <div className="record-head" style={{ marginTop: 10 }}>
        <div>
          <h1 className="page-title" style={{ margin: 0 }}>
            <StatusDot condition={l.currentConditionStatus as never} /> {l.name}
          </h1>
          <div className="muted">{l.productName} · {l.makerName} · {l.lineTypeName}</div>
        </div>
        <div className="grow" style={{ flex: 1 }} />
        <div className="record-meta">
          <div><b className={"cond-text " + condClass(l.currentConditionStatus as never)}>{l.currentConditionStatus || "—"}</b>condition</div>
          <div><b>{l.currentSide || "n/a"}</b>side in use</div>
          <div><b>{ageLabel(l.installAgeDays)}</b>install age</div>
          <div><b>{ageLabel(l.buildAgeDays)}</b>build age</div>
          <div><b>{dateLabel(l.nextInspectionDue)}</b>next inspection</div>
          <div><b><LifecycleBadge status={l.lifecycleStatus} /></b>lifecycle</div>
        </div>
      </div>

      <div className="record-meta" style={{ marginBottom: 16 }}>
        <div>Tag #&nbsp;<CopyButton value={l.tagNumber ?? ""} /></div>
        <div>Certificate #&nbsp;<CopyButton value={l.certificateNumber ?? ""} /></div>
        <div>Serial&nbsp;<CopyButton value={l.serialNumber} /></div>
      </div>

      <div className="tabs">
        {TABS.map((t) => (
          <button key={t} className={"tab" + (tab === t ? " active" : "")} onClick={() => setTab(t)}>{t}</button>
        ))}
      </div>

      {tab === "Overview" && <Overview l={l} />}
      {tab === "Side tracking" && <SideTracking l={l} />}
      {tab === "Inspections" && <InspectionsTab lineId={l.id} />}
      {tab === "Files & photos" && <FilesTab lineId={l.id} />}
    </>
  );
}

function Overview({ l }: { l: Line }) {
  return (
    <>
      <dl className="kv">
        <dt>Product</dt><dd>{l.productName}</dd>
        <dt>Maker</dt><dd>{l.makerName}</dd>
        <dt>Line type</dt><dd>{l.lineTypeName}</dd>
        <dt>Construction</dt><dd>{l.constructionType || "—"}</dd>
        <dt>SWL</dt><dd>{l.swl != null ? `${l.swl} t` : "—"}</dd>
        <dt>Break load</dt><dd>{l.breakLoad != null ? `${l.breakLoad} t` : "—"}</dd>
        <dt>Length</dt><dd>{l.length ? `${l.length} m` : "—"}</dd>
        <dt>Serial</dt><dd>{l.serialNumber}</dd>
        <dt>Location</dt><dd>{l.locationLabel}</dd>
        <dt>Manufactured</dt><dd>{dateLabel(l.manufactureDate)}</dd>
        <dt>Installed</dt><dd>{dateLabel(l.installationDate)}</dd>
        <dt>Turnable</dt><dd>{l.canBeTurned ? "Yes" : "No"}</dd>
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
                    <td>{c.name}</td><td>{c.lineTypeName}</td><td>{c.serialNumber}</td><td>{c.certificateNumber || "—"}</td>
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
  if (l.currentSide === "n/a" || !l.canBeTurned) {
    return <p className="muted">This line is not reversible — no side tracking.</p>;
  }
  const card = (side: "A" | "B", age: number, change: string | null | undefined, cond: string | undefined) => (
    <div className={"side-card" + (l.currentSide === side ? " active" : "")}>
      <h4>
        Side {side}
        {l.currentSide === side && <span className="tag-active">● in use</span>}
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
      {l.turnDue && <div className="card" style={{ borderColor: "var(--monitor)", marginBottom: 16, maxWidth: 640 }}>
        <b>Turn recommended</b> — the active side has reached the 6-month turn interval.
      </div>}
      <div className="side-cards">
        {card("A", l.sideAAgeDays, l.sideAChangeDate, l.sideACondition)}
        {card("B", l.sideBAgeDays, l.sideBChangeDate, l.sideBCondition)}
      </div>
      <div style={{ marginTop: 16, maxWidth: 640 }}>
        <TurnButton line={l} />
      </div>
    </>
  );
}
