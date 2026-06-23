// Dashboard overview hook. Raw fetch against GET /vessels/{id}/overview.
import { useQuery } from "@tanstack/react-query";
import { API_BASE } from "../../config";

export interface OverTrendPoint {
  month: string;
  inspections: number;
  action: number;
}

export interface OverAttentionItem {
  id: string;
  name: string;
  serialNumber: string;
  conditionStatus: string;
  locationLabel: string;
}

export interface OverRecentInspection {
  lineName: string;
  conditionStatus: string;
  inspectedAt: string;
}

export interface Overview {
  activeLines: number;
  spares: number;
  good: number;
  monitor: number;
  action: number;
  needingAttention: number;
  inspectionsDue: number;
  avgInstallAgeDays: number;
  attention: OverAttentionItem[];
  recentInspections: OverRecentInspection[];
  trend: OverTrendPoint[];
}

export function useOverview(vesselId?: string) {
  return useQuery({
    queryKey: ["overview", vesselId],
    enabled: !!vesselId,
    queryFn: async (): Promise<Overview> => {
      const res = await fetch(`${API_BASE}/vessels/${vesselId}/overview`);
      if (!res.ok) throw new Error(`overview ${res.status}`);
      return (await res.json()) as Overview;
    },
  });
}
