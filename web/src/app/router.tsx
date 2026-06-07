import { createBrowserRouter, Navigate } from "react-router-dom";
import { AppShell } from "./AppShell";
import { Dashboard } from "../features/dashboard/Dashboard";
import { Stub } from "../features/Stub";
import { RegisterPage } from "../features/register/RegisterPage";
import { RopeRecord } from "../features/register/RopeRecord";
import { DeckPage } from "../features/deck/DeckPage";
import { isShore } from "../config";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <Dashboard /> },
      { path: "deck", element: <DeckPage /> },
      { path: "register", element: <RegisterPage /> },
      { path: "lines/:id", element: <RopeRecord /> },
      {
        path: "inspections",
        element: <Stub title="Inspections" sub="Log inspections and view condition reports." needs="GET /lines/{id}/inspections" />,
      },
      {
        path: "logbook",
        element: <Stub title="Log book" sub="Chronological inspection log across all lines." needs="GET /inspections/logbook" />,
      },
      {
        path: "files",
        element: <Stub title="Files & certificates" sub="Condition photos, certificates and manuals." needs="GET /lines/{id}/files" />,
      },
      {
        path: "catalogue",
        element: isShore() ? (
          <Stub title="Catalogue" sub="Makers, line types and products (master data)." needs="GET /products" />
        ) : (
          <Navigate to="/" replace />
        ),
      },
      { path: "*", element: <Navigate to="/" replace /> },
    ],
  },
]);
