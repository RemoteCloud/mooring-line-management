// Permission-gating helper for write actions. Non-admin users have
// permissions.canWrite === false. <WriteGuard> renders its children and, for
// read-only users, disables any nested interactive controls and shows a
// "Read-only access" tooltip. Keep it pragmatic — wrap primary Add/Save/Delete
// actions; it does not enforce anything server-side (the backend does that).
import { cloneElement, isValidElement, type ReactElement, type ReactNode } from "react";
import { useCanWrite } from "./authContext";

const READ_ONLY_TITLE = "Read-only access";

// Wrap a single button-like element. For read-only users it is force-disabled and
// given a tooltip. Usage: <WriteGuard><button …>Add</button></WriteGuard>
export function WriteGuard({ children }: { children: ReactNode }) {
  const canWrite = useCanWrite();
  if (canWrite) return <>{children}</>;

  if (isValidElement(children)) {
    const el = children as ReactElement<{
      disabled?: boolean;
      title?: string;
      "aria-disabled"?: boolean;
    }>;
    return cloneElement(el, {
      disabled: true,
      "aria-disabled": true,
      title: READ_ONLY_TITLE,
    });
  }
  return <>{children}</>;
}
