// Settings → Users & API keys (basic, temporary auth). Raw fetch against /api, matching
// the webhooks feature's self-contained style. The global fetch wrapper attaches the key.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { API_BASE } from "../../config";

export type Role = "admin" | "vessel_user" | "readonly";

export interface Me {
  id: string;
  name: string;
  email: string;
  role: Role;
  vesselId?: string;
}

export interface User {
  id: string;
  email: string;
  name: string;
  role: Role;
  vesselId?: string;
  active: boolean;
  createdAt: string;
}

export interface ApiKey {
  id: string;
  userId: string;
  name: string;
  keyPrefix: string;
  lastUsedAt?: string;
  createdAt: string;
  revokedAt?: string;
}

export interface NewApiKey extends ApiKey {
  plainKey: string;
}

export interface CreateUserBody {
  email: string;
  name: string;
  role: Role;
  vesselId?: string;
}

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`);
  if (!res.ok) throw new Error(`GET ${path} failed: ${res.status}`);
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

// useMe powers both the unlock validation and admin-only gating of the Users panel.
export function useMe() {
  return useQuery<Me>({
    queryKey: ["me"],
    queryFn: () => getJSON<Me>("/me"),
    staleTime: 5 * 60_000,
    retry: false,
  });
}

export function useUsers(enabled: boolean) {
  return useQuery<User[]>({
    queryKey: ["users"],
    queryFn: () => getJSON<User[]>("/users"),
    enabled,
  });
}

export function useCreateUser() {
  const qc = useQueryClient();
  return useMutation<User | null, Error, CreateUserBody>({
    mutationFn: (body) => sendJSON<User>("POST", "/users", body),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["users"] }),
  });
}

export function useUpdateUser() {
  const qc = useQueryClient();
  return useMutation<User | null, Error, { id: string; active?: boolean; role?: Role }>({
    mutationFn: ({ id, ...patch }) => sendJSON<User>("PATCH", `/users/${id}`, patch),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["users"] }),
  });
}

export function useApiKeys(userId: string | null) {
  return useQuery<ApiKey[]>({
    queryKey: ["api-keys", userId],
    queryFn: () => getJSON<ApiKey[]>(`/users/${userId}/api-keys`),
    enabled: !!userId,
  });
}

export function useCreateApiKey() {
  const qc = useQueryClient();
  return useMutation<NewApiKey | null, Error, { userId: string; name: string }>({
    mutationFn: ({ userId, name }) => sendJSON<NewApiKey>("POST", `/users/${userId}/api-keys`, { name }),
    onSuccess: (_d, { userId }) => void qc.invalidateQueries({ queryKey: ["api-keys", userId] }),
  });
}

export function useRevokeApiKey() {
  const qc = useQueryClient();
  return useMutation<null, Error, { id: string; userId: string }>({
    mutationFn: ({ id }) => sendJSON<null>("DELETE", `/api-keys/${id}`),
    onSuccess: (_d, { userId }) => void qc.invalidateQueries({ queryKey: ["api-keys", userId] }),
  });
}
