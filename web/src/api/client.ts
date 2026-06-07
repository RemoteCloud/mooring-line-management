// Typed API client. openapi-fetch is fully typed against the generated schema, so
// paths, params and response bodies are checked at compile time — types come straight
// from the backend's emitted OpenAPI 3.1 spec. No business logic lives here.
import createClient from "openapi-fetch";
import type { paths } from "./schema";
import { API_BASE } from "../config";

export const api = createClient<paths>({ baseUrl: API_BASE });
