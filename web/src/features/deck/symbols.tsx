// SVG symbols for the deck map. Coordinates are normalized 0..1 and scaled to the
// viewBox by the parent, so everything stays responsive.
import type { Winch, Storage } from "../../api/hooks";
import { condClass } from "../../lib/format";

export const VB_W = 1000;
export const VB_H = 600;

function dotColor(status: string | undefined): string {
  switch (condClass(status as never)) {
    case "good": return "var(--good)";
    case "monitor": return "var(--monitor)";
    case "action": return "var(--action)";
    default: return "var(--border)";
  }
}

// Status mark mirrors the .dot CSS shapes so the deck map reads without color:
// Good=circle, Monitor=diamond, Action=triangle. cx/cy = center, r = radius.
function StatusMark({ status, cx, cy, r = 9 }: { status: string | undefined; cx: number; cy: number; r?: number }) {
  const fill = dotColor(status);
  const cls = condClass(status as never);
  const stroke = "var(--bg)";
  const sw = 1.25;
  if (cls === "monitor") {
    return <path d={`M${cx} ${cy - r}L${cx + r} ${cy}L${cx} ${cy + r}L${cx - r} ${cy}Z`} fill={fill} stroke={stroke} strokeWidth={sw} />;
  }
  if (cls === "action") {
    return <path d={`M${cx} ${cy - r}L${cx + r} ${cy + r}L${cx - r} ${cy + r}Z`} fill={fill} stroke={stroke} strokeWidth={sw} />;
  }
  return <circle cx={cx} cy={cy} r={r} fill={fill} stroke={stroke} strokeWidth={sw} />;
}

const DRUM_W = 28;
const DRUM_H = 42;
const DRUM_GAP = 5;
const PAD = 11;

export function winchBox(w: Winch) {
  const inner = w.drum_count * DRUM_W + (w.drum_count - 1) * DRUM_GAP;
  return { bw: inner + PAD * 2, bh: DRUM_H + PAD * 2, inner };
}

export function WinchSymbol({
  w, selected, highlightDrumIdx, selectedDrumIdx, onPointerDown, onClick,
}: {
  w: Winch;
  selected?: boolean;
  // 1-based drum index to highlight in accent (a specific line on this winch is deep-linked).
  highlightDrumIdx?: number;
  // 1-based drum index selected from the side panel (glows green).
  selectedDrumIdx?: number;
  onPointerDown?: (e: React.PointerEvent) => void;
  onClick?: () => void;
}) {
  const cx = w.x * VB_W;
  const cy = w.y * VB_H;
  const { bw, bh, inner } = winchBox(w);
  const byIdx = new Map(w.drums.map((d) => [d.idx, d]));
  // The label cancels the parent rotation to stay upright, so it sits straight below the
  // winch *center*. Offset by the rotated box's half-height so it clears a tilted symbol
  // instead of overlapping it (FWD-1/FWD-3 are angled).
  const rad = (w.orientation * Math.PI) / 180;
  const rotHalfH = Math.abs(Math.sin(rad)) * (bw / 2) + Math.abs(Math.cos(rad)) * (bh / 2);

  return (
    <g
      transform={`translate(${cx} ${cy}) rotate(${w.orientation})`}
      onPointerDown={onPointerDown}
      onClick={onClick}
      style={{ cursor: onPointerDown ? "grab" : "pointer" }}
    >
      <rect className={"winch-body" + (selected ? " sel" : "")} x={-bw / 2} y={-bh / 2} width={bw} height={bh} rx={8} />
      {Array.from({ length: w.drum_count }).map((_, i) => {
        const filled = (byIdx.get(i + 1)?.line_count ?? 0) > 0;
        const hl = highlightDrumIdx === i + 1;
        const sel = selectedDrumIdx === i + 1;
        const cellX = -inner / 2 + i * (DRUM_W + DRUM_GAP);
        const numCx = cellX + DRUM_W / 2;
        // Light digit on any coloured cell (filled/accent/green), dark on an empty cell.
        const numFill = filled || hl || sel ? "#fff" : "var(--text)";
        return (
          <g key={i}>
            <rect
              className={"drum-cell" + (filled ? " filled" : "") + (hl ? " hl" : "") + (sel ? " sel" : "")}
              x={cellX}
              y={-DRUM_H / 2}
              width={DRUM_W}
              height={DRUM_H}
              rx={3}
            />
            <text
              className="drum-num"
              x={numCx}
              y={0}
              fill={numFill}
              textAnchor="middle"
              dominantBaseline="central"
              transform={`rotate(${-w.orientation} ${numCx} 0)`}
            >
              {i + 1}
            </text>
          </g>
        );
      })}
      {/* worst-case status mark (shape carries status, not just color) */}
      <StatusMark status={w.worst_status} cx={bw / 2 - 7} cy={-bh / 2 + 7} />
      {/* drive-type marker, top-left: E electric / H hydraulic */}
      <text
        className="sym-drive"
        x={-bw / 2 + 9}
        y={-bh / 2 + 15}
        transform={`rotate(${-w.orientation} ${-bw / 2 + 9} ${-bh / 2 + 11})`}
      >
        {w.drive_type === "hydraulic" ? "H" : "E"}
      </text>
      <text className="sym-label" x={0} y={rotHalfH + 16} transform={`rotate(${-w.orientation})`}>{w.label}</text>
    </g>
  );
}

export function StorageSymbol({
  s, selected, highlighted, onPointerDown, onClick,
}: {
  s: Storage;
  selected?: boolean;
  // true when the selected line is stored here.
  highlighted?: boolean;
  onPointerDown?: (e: React.PointerEvent) => void;
  onClick?: () => void;
}) {
  const cx = s.x * VB_W;
  const cy = s.y * VB_H;
  const w = 70, h = 46;
  return (
    <g transform={`translate(${cx} ${cy})`} onPointerDown={onPointerDown} onClick={onClick}
       style={{ cursor: onPointerDown ? "grab" : "pointer" }}>
      <rect className={"winch-body" + (selected ? " sel" : "") + (highlighted ? " hl" : "")} x={-w / 2} y={-h / 2} width={w} height={h} rx={6}
            strokeDasharray="5 4" />
      <text className="sym-label" x={0} y={4}>▤ {s.line_count}</text>
      <StatusMark status={s.worst_status} cx={w / 2 - 4} cy={-h / 2 + 4} />
      <text className="sym-label" x={0} y={h / 2 + 16}>{s.label}</text>
    </g>
  );
}

export function Hull({ station }: { station: "fwd" | "aft" }) {
  // Forward deck has a pointed bow at the top; aft deck a flat stern.
  const pts =
    station === "fwd"
      ? `${VB_W / 2},20 850,170 850,560 150,560 150,170`
      : `150,40 850,40 850,470 640,580 360,580 150,470`;
  return <polygon className="hull" points={pts} />;
}
