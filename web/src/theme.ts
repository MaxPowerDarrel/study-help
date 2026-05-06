import { useSyncExternalStore } from "react";
import { defaultToggleStore, ToggleStore } from "./platform/ToggleStore";

export type ThemeChoice = "light" | "dark" | "system";

const KEY = "theme";

export function readTheme(
  store: ToggleStore = defaultToggleStore,
): ThemeChoice {
  const v = store.get(KEY);
  if (v === "light" || v === "dark") return v;
  return "system";
}

export function applyTheme(choice: ThemeChoice): void {
  const root = document.documentElement;
  if (choice === "system") {
    root.removeAttribute("data-theme");
  } else {
    root.setAttribute("data-theme", choice);
  }
}

export function writeTheme(
  choice: ThemeChoice,
  store: ToggleStore = defaultToggleStore,
): void {
  if (choice === "system") {
    store.remove(KEY);
  } else {
    store.set(KEY, choice);
  }
  applyTheme(choice);
}

export function useTheme(): [ThemeChoice, (c: ThemeChoice) => void] {
  const raw = useSyncExternalStore(
    (cb) => defaultToggleStore.subscribe(cb),
    () => defaultToggleStore.get(KEY),
    () => null,
  );
  const choice: ThemeChoice =
    raw === "light" || raw === "dark" ? raw : "system";
  return [choice, (c) => writeTheme(c)];
}
