import { createBrowserRouter, Navigate } from "react-router-dom";
import { AppShell } from "./AppShell";
import { Dashboard } from "../features/dashboard/Dashboard";
import { Stub } from "../features/Stub";
import { RegisterPage } from "../features/register/RegisterPage";
import { RopeRecord } from "../features/register/RopeRecord";
import { DeckPage } from "../features/deck/DeckPage";
import { InspectionsPage } from "../features/inspections/InspectionsPage";
import { LogbookPage } from "../features/inspections/LogbookPage";
import { CataloguePage } from "../features/catalogue/CataloguePage";
import { RequireAuth } from "./auth/RequireAuth";

export const router = createBrowserRouter([
  {
    path: "/",
    element: (
      <RequireAuth>
        <AppShell />
      </RequireAuth>
    ),
    children: [
      { index: true, element: <Dashboard /> },
      { path: "deck", element: <DeckPage /> },
      { path: "register", element: <RegisterPage /> },
      { path: "lines/:id", element: <RopeRecord /> },
      { path: "inspections", element: <InspectionsPage /> },
      { path: "logbook", element: <LogbookPage /> },
      {
        path: "files",
        element: <Stub title="Files & certificates" sub="Condition photos, certificates and manuals live on each rope record's Files & photos tab." needs="open a line → Files & photos" />,
      },
      {
        path: "catalogue",
        element: <CataloguePage />,
      },
      { path: "*", element: <Navigate to="/" replace /> },
    ],
  },
]);
