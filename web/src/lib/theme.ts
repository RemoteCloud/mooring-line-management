// Theme is stored on <html data-theme> (set pre-paint in index.html to avoid a
// flash) and persisted in localStorage. Default follows the OS preference.
export type Theme = "light" | "dark";

export function getTheme(): Theme {
  return document.documentElement.dataset.theme === "light" ? "light" : "dark";
}

export function setTheme(t: Theme): void {
  document.documentElement.dataset.theme = t;
  try {
    localStorage.setItem("theme", t);
  } catch {
    /* private mode / storage disabled — in-memory only */
  }
}

export function toggleTheme(): Theme {
  const next: Theme = getTheme() === "dark" ? "light" : "dark";
  setTheme(next);
  return next;
}
