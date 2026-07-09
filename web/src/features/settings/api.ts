// Settings → Webhooks API hooks. Raw fetch against the same-origin /api prefix,
// matching the catalogue feature's self-contained style.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { API_BASE } from "../../config";

export interface WebhookSubscription {
  id: string;
  vesselId?: string;
  name: string;
  url: string;
  events: string[];
  headers: Record<string, string>;
  payloadTemplate?: string;
  active: boolean;
  hasSecret: boolean;
  createdAt: string;
}

export interface WebhookEvent {
  type: string;
  description: string;
  variables: string[];
}

export interface WebhookInput {
  name: string;
  url: string;
  secret?: string;
  events: string[];
  headers: Record<string, string>;
  payloadTemplate?: string;
  active: boolean;
}

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`);
  if (!res.ok) throw new Error(`GET ${path} failed: ${res.status} ${res.statusText}`);
  return (await res.json()) as T;
}

async function sendJSON<T>(method: string, path: string, body?: unknown): Promise<T | null> {
  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers: body === undefined ? undefined : { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (!res.ok) {
    let detail = `${res.status} ${res.statusText}`;
    try {
      const j = await res.json();
      if (j?.detail) detail = j.detail;
    } catch {
      /* non-JSON error body */
    }
    throw new Error(detail);
  }
  if (res.status === 204) return null;
  return (await res.json()) as T;
}

const vq = (vesselId?: string) => (vesselId ? `?vesselId=${encodeURIComponent(vesselId)}` : "");

export function useWebhookEvents() {
  return useQuery<WebhookEvent[]>({
    queryKey: ["webhook-events"],
    queryFn: () => getJSON<WebhookEvent[]>("/webhook-events"),
    staleTime: Infinity,
  });
}

export function useWebhooks(vesselId?: string) {
  return useQuery<WebhookSubscription[]>({
    queryKey: ["webhooks", vesselId],
    queryFn: () => getJSON<WebhookSubscription[]>(`/webhooks${vq(vesselId)}`),
  });
}

export function useCreateWebhook(vesselId?: string) {
  const qc = useQueryClient();
  return useMutation<WebhookSubscription | null, Error, WebhookInput>({
    mutationFn: (body) => sendJSON<WebhookSubscription>("POST", `/webhooks${vq(vesselId)}`, body),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["webhooks", vesselId] }),
  });
}

export function useUpdateWebhook(vesselId?: string) {
  const qc = useQueryClient();
  return useMutation<WebhookSubscription | null, Error, { id: string; body: WebhookInput }>({
    mutationFn: ({ id, body }) => sendJSON<WebhookSubscription>("PUT", `/webhooks/${id}`, body),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["webhooks", vesselId] }),
  });
}

export function useDeleteWebhook(vesselId?: string) {
  const qc = useQueryClient();
  return useMutation<null, Error, string>({
    mutationFn: (id) => sendJSON<null>("DELETE", `/webhooks/${id}`),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["webhooks", vesselId] }),
  });
}

export function useTestWebhook() {
  return useMutation<{ ok: boolean } | null, Error, string>({
    mutationFn: (id) => sendJSON<{ ok: boolean }>("POST", `/webhooks/${id}/test`),
  });
}
