// Vessel overview dashboard (OV-1/2/3). KPI tiles + condition donut + attention
// list + 5-month trend, wired to GET /vessels/{id}/overview.
import { Link } from "react-router-dom";
import { useVessel } from "../../app/VesselContext";
import { StatusDot } from "../../components/ui";
import { ageLabel, dateLabel, condClass, type Condition } from "../../lib/format";
import { useOverview, type Overview, type OverTrendPoint } from "./api";
import "./dashboard.css";

export function Dashboard() {
  const { vesselId } = useVessel();
  const { data, isLoading } = useOverview(vesselId);

  return (
    <>
      <h1 className="page-title">Vessel overview</h1>
      <p className="page-sub">Condition summary and lines needing attention.</p>

      {!vesselId ? (
        <div className="stub">Select a vessel to see its overview.</div>
      ) : isLoading || !data ? (
        <div className="stub">Loading overview…</div>
      ) : (
        <OverviewBody o={data} />
      )}
    </>
  );
}

function OverviewBody({ o }: { o: Overview }) {
  return (
    <>
      <div className="cards">
        <div className="card">
          <div className="kpi-label">Active lines</div>
          <div className="kpi-value">{o.active_lines}</div>
        </div>
        <div className="card">
          <div className="kpi-label">Needing attention</div>
          <div className="kpi-value">{o.needing_attention}</div>
        </div>
        <div className="card">
          <div className="kpi-label">Inspections due</div>
          <div className="kpi-value">{o.inspections_due}</div>
        </div>
        <div className="card">
          <div className="kpi-label">Avg install age</div>
          <div className="kpi-value">{ageLabel(o.avg_install_age_days)}</div>
        </div>
      </div>

      <div className="dash-grid">
        <div className="panel">
          <h3>Condition</h3>
          <ConditionDonut good={o.good} monitor={o.monitor} action={o.action} />
        </div>

        <div className="panel">
          <h3>Needs attention</h3>
          <AttentionList o={o} />
        </div>
      </div>

      <div className="dash-grid">
        <div className="panel">
          <h3>Inspections — last 5 months</h3>
          <TrendChart points={o.trend} />
        </div>

        <div className="panel">
          <h3>Recent inspections</h3>
          <RecentTable o={o} />
        </div>
      </div>
    </>
  );
}

function ConditionDonut({ good, monitor, action }: { good: number; monitor: number; action: number }) {
  const total = good + monitor + action;
  const size = 140;
  const r = 56;
  const cx = size / 2;
  const cy = size / 2;
  const circ = 2 * Math.PI * r;
  const stroke = 16;

  const segs: { value: number; color: string }[] = [
    { value: good, color: "var(--good)" },
    { value: monitor, color: "var(--monitor)" },
    { value: action, color: "var(--action)" },
  ];

  let offset = 0;
  return (
    <div className="donut-wrap">
      <svg className="donut" width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
        <circle
          className="donut-ring-bg"
          cx={cx}
          cy={cy}
          r={r}
          fill="none"
          strokeWidth={stroke}
        />
        {total > 0 &&
          segs.map((s, i) => {
            if (s.value <= 0) return null;
            const len = (s.value / total) * circ;
            const el = (
              <circle
                key={i}
                cx={cx}
                cy={cy}
                r={r}
                fill="none"
                stroke={s.color}
                strokeWidth={stroke}
                strokeDasharray={`${len} ${circ - len}`}
                strokeDashoffset={-offset}
                transform={`rotate(-90 ${cx} ${cy})`}
              />
            );
            offset += len;
            return el;
          })}
        <text className="donut-center" x={cx} y={cy + 2} textAnchor="middle" dominantBaseline="middle">
          {total}
        </text>
        <text className="donut-sub" x={cx} y={cy + 16} textAnchor="middle">
          LINES
        </text>
      </svg>
      <div className="legend">
        <LegendRow color="var(--good)" name="Good" count={good} />
        <LegendRow color="var(--monitor)" name="Monitor" count={monitor} />
        <LegendRow color="var(--action)" name="Action" count={action} />
      </div>
    </div>
  );
}

function LegendRow({ color, name, count }: { color: string; name: string; count: number }) {
  return (
    <div className="legend-row">
      <span className="swatch" style={{ background: color }} />
      <span className="legend-name">{name}</span>
      <span className="legend-count">{count}</span>
    </div>
  );
}

function AttentionList({ o }: { o: Overview }) {
  if (o.attention.length === 0) {
    return <div className="muted">No lines need attention.</div>;
  }
  return (
    <div className="attn-list">
      {o.attention.map((it) => (
        <Link key={it.id} to={`/lines/${it.id}`} className="attn-item">
          <StatusDot condition={it.condition_status as Condition} />
          <span className="attn-main">
            <span className="attn-name">{it.name}</span>
            <span className="attn-meta">{it.serial_number}</span>
          </span>
          <span className="attn-loc">{it.location_label}</span>
          <span className={"attn-cond " + condClass(it.condition_status as Condition)}>
            {it.condition_status}
          </span>
        </Link>
      ))}
    </div>
  );
}

function TrendChart({ points }: { points: OverTrendPoint[] }) {
  const max = Math.max(1, ...points.map((p) => p.inspections));
  return (
    <div className="trend">
      {points.map((p) => {
        const hPct = (p.inspections / max) * 100;
        const actionPct = p.inspections > 0 ? (p.action / p.inspections) * 100 : 0;
        return (
          <div className="trend-col" key={p.month}>
            <span className="trend-val">{p.inspections || ""}</span>
            <div className="trend-bar-track">
              <div
                className={"trend-bar" + (p.action > 0 ? " has-action" : "")}
                style={{ height: `${hPct}%` }}
                title={`${p.month}: ${p.inspections} inspections, ${p.action} action`}
              >
                {p.action > 0 && <div className="trend-action" style={{ height: `${actionPct}%` }} />}
              </div>
            </div>
            <span className="trend-label">{p.month}</span>
          </div>
        );
      })}
    </div>
  );
}

function RecentTable({ o }: { o: Overview }) {
  if (o.recent_inspections.length === 0) {
    return <div className="muted">No inspections recorded.</div>;
  }
  return (
    <table className="recent-table">
      <tbody>
        {o.recent_inspections.map((ri, i) => (
          <tr key={i}>
            <td>{ri.line_name}</td>
            <td>
              <span className="recent-cond">
                <StatusDot condition={ri.condition_status as Condition} />
                {ri.condition_status || "—"}
              </span>
            </td>
            <td className="muted">{dateLabel(ri.inspected_at)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
