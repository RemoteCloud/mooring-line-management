// Vessel switcher (XC-1). Shore only — onboard serves a single vessel.
import { isOnboard } from "../config";
import { useVessel } from "./VesselContext";

export function VesselSwitcher() {
  const { vessels, vesselId, setVesselId } = useVessel();
  if (isOnboard()) return null;
  return (
    <select
      className="vessel-switch"
      value={vesselId ?? ""}
      onChange={(e) => setVesselId(e.target.value)}
      aria-label="Select vessel"
    >
      {vessels.map((v) => (
        <option key={v.id} value={v.id}>
          {v.name}
        </option>
      ))}
    </select>
  );
}
