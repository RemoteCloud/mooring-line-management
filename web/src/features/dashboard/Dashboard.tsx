// Vessel overview dashboard (OV-1/2/3). KPI tiles + condition donut + attention list
// + trend wire up to GET /vessels/{id}/overview once the dashboard slice lands.
// For now the connectivity tile is live (real /health); the rest are placeholders.
import { useHealth } from "../../api/hooks";

export function Dashboard() {
  const { data } = useHealth();

  return (
    <>
      <h1 className="page-title">Fleet overview</h1>
      <p className="page-sub">Condition summary and lines needing attention.</p>

      <div className="cards">
        <div className="card">
          <div className="kpi-label">Active lines</div>
          <div className="kpi-value">—</div>
        </div>
        <div className="card">
          <div className="kpi-label">Needing attention</div>
          <div className="kpi-value">—</div>
        </div>
        <div className="card">
          <div className="kpi-label">Inspections due</div>
          <div className="kpi-value">—</div>
        </div>
        <div className="card">
          <div className="kpi-label">API status</div>
          <div className="kpi-value" style={{ fontSize: 20, marginTop: 12 }}>
            {data ? `${data.scope} · db ${data.db}` : "connecting…"}
          </div>
        </div>
      </div>

      <div style={{ marginTop: 24 }} className="stub">
        <h3>Condition donut, attention list &amp; 5-month trend</h3>
        Renders here once the dashboard API slice (GET /vessels/&#123;id&#125;/overview) lands.
      </div>
    </>
  );
}
