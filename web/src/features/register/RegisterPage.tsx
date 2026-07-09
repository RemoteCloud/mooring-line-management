import { useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useVessel } from "../../app/VesselContext";
import { useLines, useLineTypes, type LineRow, type LineFilters } from "../../api/hooks";
import { StatusDot, LifecycleBadge } from "../../components/ui";
import { ageLabel } from "../../lib/format";
import { AddLineDialog } from "./AddLineDialog";

type SortKey = keyof Pick<
  LineRow,
  "name" | "productName" | "makerName" | "lineTypeName" | "locationLabel" |
  "currentConditionStatus" | "currentSide" | "installAgeDays" | "lifecycleStatus"
>;

const COLUMNS: { key: SortKey; label: string }[] = [
  { key: "name", label: "Line" },
  { key: "productName", label: "Product" },
  { key: "lineTypeName", label: "Type" },
  { key: "locationLabel", label: "Location" },
  { key: "currentConditionStatus", label: "Condition" },
  { key: "currentSide", label: "Side" },
  { key: "installAgeDays", label: "Install age" },
  { key: "lifecycleStatus", label: "Status" },
];

export function RegisterPage() {
  const { vesselId } = useVessel();
  const [params, setParams] = useSearchParams();
  const navigate = useNavigate();
  const [addOpen, setAddOpen] = useState(false);

  const q = params.get("q") ?? "";
  const [condition, setCondition] = useState<LineFilters["condition"]>();
  const [lineTypeId, setLineTypeId] = useState<string>("");
  const [placement, setPlacement] = useState<LineFilters["placement"] | "">("");
  const [sort, setSort] = useState<{ key: SortKey; dir: 1 | -1 }>({ key: "name", dir: 1 });

  const { data: lineTypes = [] } = useLineTypes();
  const filters: LineFilters = {
    q: q || undefined,
    condition,
    lineTypeId: lineTypeId || undefined,
    placement: placement || undefined,
  };
  const { data, isLoading } = useLines(vesselId, filters);

  const rows = useMemo(() => {
    const items = [...(data?.items ?? [])];
    items.sort((a, b) => {
      const av = a[sort.key] ?? "";
      const bv = b[sort.key] ?? "";
      if (typeof av === "number" && typeof bv === "number") return (av - bv) * sort.dir;
      return String(av).localeCompare(String(bv)) * sort.dir;
    });
    return items;
  }, [data, sort]);

  return (
    <>
      <div className="toolbar">
        <h1 className="page-title" style={{ margin: 0 }}>Rope register</h1>
        <div className="grow" />
        <button className="btn" onClick={() => setAddOpen(true)}>+ Add line</button>
      </div>

      <div className="toolbar">
        <input
          className="input"
          placeholder="Search ID, serial, product, location…"
          value={q}
          onChange={(e) => setParams(e.target.value ? { q: e.target.value } : {})}
          style={{ minWidth: 260 }}
        />
        <select className="input" value={lineTypeId} onChange={(e) => setLineTypeId(e.target.value)}>
          <option value="">All types</option>
          {lineTypes.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
        </select>
        <select
          className="input"
          value={condition ?? ""}
          onChange={(e) => setCondition((e.target.value || undefined) as LineFilters["condition"])}
        >
          <option value="">All conditions</option>
          <option>Good</option><option>Monitor</option><option>Action</option>
        </select>
        <div className="seg">
          {(["", "installed", "spare"] as const).map((p) => (
            <button key={p} className={placement === p ? "active" : ""} onClick={() => setPlacement(p)}>
              {p === "" ? "All" : p}
            </button>
          ))}
        </div>
        <div className="grow" />
        <span className="count">{isLoading ? "loading…" : `${rows.length} of ${data?.total ?? 0} lines`}</span>
      </div>

      <div className="table-wrap">
        <table className="grid">
          <thead>
            <tr>
              {COLUMNS.map((c) => (
                <th
                  key={c.key}
                  onClick={() =>
                    setSort((s) => ({ key: c.key, dir: s.key === c.key && s.dir === 1 ? -1 : 1 }))
                  }
                >
                  {c.label}{sort.key === c.key ? (sort.dir === 1 ? " ▲" : " ▼") : ""}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.id} onClick={() => navigate(`/lines/${r.id}`)}>
                <td>
                  <div>{r.name}</div>
                  <div className="muted" style={{ fontSize: 12 }}>{r.serialNumber}</div>
                </td>
                <td>
                  <div>{r.productName}</div>
                  <div className="muted" style={{ fontSize: 12 }}>{r.makerName}</div>
                </td>
                <td>{r.lineTypeName}</td>
                <td>{r.locationLabel}</td>
                <td><StatusDot condition={r.currentConditionStatus as never} /> {r.currentConditionStatus || "—"}</td>
                <td>{r.currentSide || "—"}</td>
                <td>{ageLabel(r.installAgeDays)}</td>
                <td><LifecycleBadge status={r.lifecycleStatus} /></td>
              </tr>
            ))}
            {rows.length === 0 && !isLoading && (
              <tr><td colSpan={COLUMNS.length} className="muted" style={{ textAlign: "center", padding: 30 }}>No lines match.</td></tr>
            )}
          </tbody>
        </table>
      </div>

      {addOpen && vesselId && <AddLineDialog vesselId={vesselId} onClose={() => setAddOpen(false)} />}
    </>
  );
}
