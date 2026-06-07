// Resolves the vessel a request targets. Onboard is pinned to its configured vessel;
// shore lets the user switch. Provides the current vessel id to all features.
import { createContext, useContext, useMemo, useState, type ReactNode } from "react";
import { isOnboard, ONBOARD_VESSEL_ID } from "../config";
import { useVessels, type Vessel } from "../api/hooks";

type Ctx = {
  vesselId: string | undefined;
  vessels: Vessel[];
  setVesselId: (id: string) => void;
};

const VesselCtx = createContext<Ctx>({ vesselId: undefined, vessels: [], setVesselId: () => {} });

export function VesselProvider({ children }: { children: ReactNode }) {
  const { data: vessels = [] } = useVessels();
  const [selected, setSelected] = useState<string | undefined>();

  const vesselId = useMemo(() => {
    if (isOnboard()) return ONBOARD_VESSEL_ID;
    return selected ?? vessels[0]?.id;
  }, [selected, vessels]);

  return (
    <VesselCtx.Provider value={{ vesselId, vessels, setVesselId: setSelected }}>
      {children}
    </VesselCtx.Provider>
  );
}

export const useVessel = () => useContext(VesselCtx);
