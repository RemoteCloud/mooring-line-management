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
  serial_number: string;
  condition_status: string;
  location_label: string;
}

export interface OverRecentInspection {
  line_name: string;
  condition_status: string;
  inspected_at: string;
}

export interface Overview {
  active_lines: number;
  spares: number;
  good: number;
  monitor: number;
  action: number;
  needing_attention: number;
  inspections_due: number;
  avg_install_age_days: number;
  attention: OverAttentionItem[];
  recent_inspections: OverRecentInspection[];
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
