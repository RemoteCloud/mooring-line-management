// Typed API client. openapi-fetch is fully typed against the generated schema, so
// paths, params and response bodies are checked at compile time — types come straight
// from the backend's emitted OpenAPI 3.1 spec. No business logic lives here.
import createClient, { type Middleware } from "openapi-fetch";
import type { paths } from "./schema";
import { API_BASE } from "../config";
import { getApiKey, clearApiKey } from "./authKey";

// openapi-fetch binds globalThis.fetch when the client is created (at import time),
// which runs before installAuthFetch() patches window.fetch — so the global wrapper does
// NOT cover the typed client. Attach the API key here at request time instead, and drop
// to the unlock gate on a 401.
const authMiddleware: Middleware = {
  onRequest({ request }) {
    const key = getApiKey();
    if (key) request.headers.set("Authorization", `Bearer ${key}`);
    return request;
  },
  onResponse({ response }) {
    if (response.status === 401) clearApiKey();
    return response;
  },
};

export const api = createClient<paths>({ baseUrl: API_BASE });
api.use(authMiddleware);
