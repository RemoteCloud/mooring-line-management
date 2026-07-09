import { useTurnLine } from "./useTurnLine";

type TurnableLine = {
  id: string;
  canBeTurned: boolean;
  currentSide?: string;
  currentConditionStatus?: string;
};

// TurnButton flips a line to its other side. Disabled unless the line is
// turnable and currently installed on a definite side (A or B).
export function TurnButton({ line }: { line: TurnableLine }) {
  const turn = useTurnLine(line.id);

  const disabled =
    !line.canBeTurned ||
    line.currentSide === "n/a" ||
    !line.currentSide;

  function onClick() {
    if (turn.isPending) return;
    if (!window.confirm("Turn this line to its other side?")) return;
    turn.mutate({});
  }

  return (
    <div className="card">
      <button
        className="btn"
        disabled={disabled || turn.isPending}
        onClick={onClick}
      >
        {turn.isPending ? "Turning…" : "Turn to other side"}
      </button>
      {disabled && (
        <p className="muted">This line cannot be turned.</p>
      )}
      {turn.isError && (
        <p className="muted">
          {turn.error instanceof Error ? turn.error.message : "Turn failed."}
        </p>
      )}
    </div>
  );
}
