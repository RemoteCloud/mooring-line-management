import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider } from "react-router-dom";
import { queryClient } from "./api/queryClient";
import { router } from "./app/router";
import { installAuthFetch } from "./api/authKey";
import { UnlockGate } from "./app/UnlockGate";
import "./styles.css";

// Attach the API key to every /api request before anything renders.
installAuthFetch();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <UnlockGate>
        <RouterProvider router={router} />
      </UnlockGate>
    </QueryClientProvider>
  </StrictMode>
);
