// View-mode rope-management panel for a selected winch (or storage). Lists the
// winch's drums and lets the crew assign / register / move / turn / inspect the
// ropes on them without leaving the deck map. Drum↔rope matching is by
// currentDrumId (not label parsing).
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  useMoveLine, type Layout, type Winch, type Storage, type LineRow,
} from "../../api/hooks";
import { useTurnLine } from "../turning/useTurnLine";
import { StatusDot } from "../../components/ui";
import { LogInspectionDialog } from "../inspections/LogInspectionDialog";
import { AddLineDialog } from "../register/AddLineDialog";

type Dialog =
  | { kind: "move"; line: LineRow }
  | { kind: "assign"; drumId: string; drumLabel: string }
  | { kind: "register"; drumId: string; drumLabel: string }
  | { kind: "inspect"; line: LineRow };

// A line is assignable to a drum if it's off every drum and is either a spare or
// an active rope currently sitting in storage (being redeployed). Excludes
// ordered / retired / active-but-nowhere.
function isAssignable(l: LineRow): boolean {
  if (l.currentDrumId) return false;
  if (l.lifecycleStatus === "spare") return true;
  return l.lifecycleStatus === "active" && !!l.currentStorageId;
}

export function WinchPanel({
  winch, layout, lines, vesselId, selectedDrumIdx, onSelectDrum,
}: {
  winch: Winch;
  layout: Layout;
  lines: LineRow[];
  vesselId: string;
  // 1-based drum index selected from this panel; glows the matching cell on the map.
  selectedDrumIdx: number | null;
  onSelectDrum: (idx: number) => void;
}) {
  const [dialog, setDialog] = useState<Dialog | null>(null);
  const drums = [...(winch.drums ?? [])].sort((a, b) => a.idx - b.idx);
  const lineOnDrum = (drumId: string) => lines.find((l) => l.currentDrumId === drumId);

  return (
    <>
      <h3 style={{ marginTop: 0 }}>{winch.label}</h3>
      <p className="muted" style={{ marginTop: -6 }}>
        {winch.driveType === "hydraulic" ? "Hydraulic" : "Electric"} · {drums.length} drum{drums.length === 1 ? "" : "s"}
        {winch.swl != null && ` · SWL ${winch.swl} t`}
        {winch.breakLoad != null && ` · MBL ${winch.breakLoad} t`}
      </p>

      <div className="drum-rows">
        {drums.map((d) => {
          const line = lineOnDrum(d.id);
          const label = `${winch.label} · D${d.idx}`;
          const sel = selectedDrumIdx === d.idx;
          return (
            <div key={d.id} className={"drum-row" + (sel ? " sel" : "")}>
              {line ? (
                <>
                  <button className="drum-head drum-select" onClick={() => onSelectDrum(d.idx)} aria-pressed={sel}>
                    <span className="drum-tag">D{d.idx}</span>
                    <span className="drum-line">
                      <StatusDot condition={line.currentConditionStatus as never} />
                      <span className="drum-line-name">{line.name}</span>
                      <span className="muted drum-line-side">Side {line.currentSide || "—"}</span>
                    </span>
                  </button>
                  <div className="drum-actions">
                    <button className="chip" onClick={() => setDialog({ kind: "move", line })}>Move</button>
                    <TurnChip line={line} />
                    <button className="chip" onClick={() => setDialog({ kind: "inspect", line })}>Inspect</button>
                  </div>
                </>
              ) : (
                <>
                  <button className="drum-head drum-select" onClick={() => onSelectDrum(d.idx)} aria-pressed={sel}>
                    <span className="drum-tag">D{d.idx}</span>
                    <span className="muted drum-empty">empty</span>
                  </button>
                  <div className="drum-actions">
                    <button className="chip" onClick={() => setDialog({ kind: "assign", drumId: d.id, drumLabel: label })}>Assign</button>
                    <button className="chip" onClick={() => setDialog({ kind: "register", drumId: d.id, drumLabel: label })}>Register here</button>
                  </div>
                </>
              )}
            </div>
          );
        })}
      </div>

      {dialog?.kind === "move" && (
        <MovePicker line={dialog.line} layout={layout} lines={lines} vesselId={vesselId} onClose={() => setDialog(null)} />
      )}
      {dialog?.kind === "assign" && (
        <AssignPicker drumId={dialog.drumId} drumLabel={dialog.drumLabel} lines={lines} vesselId={vesselId} onClose={() => setDialog(null)} />
      )}
      {dialog?.kind === "register" && (
        <AddLineDialog vesselId={vesselId} targetDrumId={dialog.drumId} targetLabel={dialog.drumLabel} onClose={() => setDialog(null)} />
      )}
      {dialog?.kind === "inspect" && (
        <LogInspectionDialog lineId={dialog.line.id} onClose={() => setDialog(null)} />
      )}
    </>
  );
}

// StoragePanel mirrors the winch panel for ropes in a storage location.
export function StoragePanel({
  storage, layout, lines, vesselId,
}: {
  storage: Storage;
  layout: Layout;
  lines: LineRow[];
  vesselId: string;
}) {
  const navigate = useNavigate();
  const [dialog, setDialog] = useState<Dialog | null>(null);
  const here = lines.filter((l) => l.currentStorageId === storage.id);

  return (
    <>
      <h3 style={{ marginTop: 0 }}>{storage.label}</h3>
      <p className="muted" style={{ marginTop: -6 }}>{here.length} rope{here.length === 1 ? "" : "s"} stored</p>
      {here.length === 0 && <p className="muted">No ropes here.</p>}
      <div className="drum-rows">
        {here.map((line) => (
          <div key={line.id} className="drum-row">
            <button className="drum-line" onClick={() => navigate(`/lines/${line.id}`)}>
              <StatusDot condition={line.currentConditionStatus as never} />
              <span className="drum-line-name">{line.name}</span>
              <span className="muted drum-line-side">{line.serialNumber}</span>
            </button>
            <div className="drum-actions">
              <button className="chip" onClick={() => setDialog({ kind: "move", line })}>Move</button>
              <button className="chip" onClick={() => setDialog({ kind: "inspect", line })}>Inspect</button>
            </div>
          </div>
        ))}
      </div>
      {dialog?.kind === "move" && (
        <MovePicker line={dialog.line} layout={layout} lines={lines} vesselId={vesselId} onClose={() => setDialog(null)} />
      )}
      {dialog?.kind === "inspect" && (
        <LogInspectionDialog lineId={dialog.line.id} onClose={() => setDialog(null)} />
      )}
    </>
  );
}

function TurnChip({ line }: { line: LineRow }) {
  const turn = useTurnLine(line.id);
  // LineRow doesn't carry canBeTurned; allow turning whenever the rope is on a
  // definite side and let the backend reject non-turnable lines.
  const disabled = !line.currentSide || line.currentSide === "n/a" || turn.isPending;
  const onClick = () => {
    if (turn.isPending) return;
    if (!window.confirm(`Turn ${line.name} to its other side?`)) return;
    turn.mutate({});
  };
  return (
    <button className="chip" disabled={disabled} onClick={onClick} title={disabled ? "Needs a definite side (A/B)" : "Turn side"}>
      {turn.isPending ? "Turning…" : "Turn"}
    </button>
  );
}

// MovePicker lists valid destinations on the same vessel: empty drums (any
// winch) plus storage. Excludes the line's current drum.
function MovePicker({
  line, layout, lines, vesselId, onClose,
}: {
  line: LineRow;
  layout: Layout;
  lines: LineRow[];
  vesselId: string;
  onClose: () => void;
}) {
  const move = useMoveLine(vesselId);
  const occupied = new Set(lines.map((l) => l.currentDrumId).filter(Boolean) as string[]);

  const drumTargets = (layout.winches ?? []).flatMap((w) =>
    [...(w.drums ?? [])].sort((a, b) => a.idx - b.idx)
      .filter((d) => !occupied.has(d.id) && d.id !== line.currentDrumId)
      .map((d) => ({ id: d.id, kind: "drum" as const, label: `${w.label} · D${d.idx}`, station: w.station })),
  );
  const storageTargets = (layout.storage ?? [])
    .filter((s) => s.id !== line.currentStorageId)
    .map((s) => ({ id: s.id, kind: "storage" as const, label: s.label, station: s.station }));
  const targets = [...drumTargets, ...storageTargets];

  const go = (t: { id: string; kind: "drum" | "storage" }) => {
    move.mutate(
      { lineId: line.id, ...(t.kind === "drum" ? { toDrumId: t.id } : { toStorageId: t.id }) },
      { onSuccess: onClose },
    );
  };

  return (
    <div className="overlay" onClick={onClose}>
      <div className="dialog" onClick={(e) => e.stopPropagation()}>
        <h3>Move {line.name}</h3>
        <p className="muted" style={{ marginTop: -6 }}>Pick an empty drum or a storage location.</p>
        {move.isError && <div className="err">{move.error?.message ?? "Move failed."}</div>}
        <div className="pick-list">
          {targets.length === 0 && <p className="muted">No free destinations.</p>}
          {targets.map((t) => (
            <button key={t.id} className="pick-row" disabled={move.isPending} onClick={() => go(t)}>
              <span>{t.label}</span>
              <span className="muted">{t.kind === "drum" ? `Drum · ${t.station}` : t.station ? `Storage · ${t.station}` : "Storage area"}</span>
            </button>
          ))}
        </div>
        <div className="dialog-actions">
          <button className="btn ghost" onClick={onClose}>Cancel</button>
        </div>
      </div>
    </div>
  );
}

// AssignPicker lists assignable spares/storage ropes to drop onto an empty drum.
function AssignPicker({
  drumId, drumLabel, lines, vesselId, onClose,
}: {
  drumId: string;
  drumLabel: string;
  lines: LineRow[];
  vesselId: string;
  onClose: () => void;
}) {
  const move = useMoveLine(vesselId);
  const candidates = lines.filter(isAssignable);

  const go = (lineId: string) => move.mutate({ lineId, toDrumId: drumId }, { onSuccess: onClose });

  return (
    <div className="overlay" onClick={onClose}>
      <div className="dialog" onClick={(e) => e.stopPropagation()}>
        <h3>Assign rope to {drumLabel}</h3>
        {move.isError && <div className="err">{move.error?.message ?? "Assign failed."}</div>}
        <div className="pick-list">
          {candidates.length === 0 && <p className="muted">No spare or stored ropes available. Use “Register here” to add a new one.</p>}
          {candidates.map((l) => (
            <button key={l.id} className="pick-row" disabled={move.isPending} onClick={() => go(l.id)}>
              <span>
                <StatusDot condition={l.currentConditionStatus as never} /> {l.name}
              </span>
              <span className="muted">{l.lifecycleStatus === "spare" ? "Spare" : "In storage"} · {l.serialNumber}</span>
            </button>
          ))}
        </div>
        <div className="dialog-actions">
          <button className="btn ghost" onClick={onClose}>Cancel</button>
        </div>
      </div>
    </div>
  );
}
