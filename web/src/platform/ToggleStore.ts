// ToggleStore is the platform abstraction for formatting-toggle
// persistence (PROJECT_CONSTITUTION.md §4 — platform features behind an
// abstraction, and passage-reader.md Decisions, 2026-05-03).
//
// A future native shell (e.g. iPad WebView wrapper) can substitute its
// own implementation without touching the reading-surface component.
export interface ToggleStore {
  get(key: string): string | null;
  set(key: string, value: string): void;
  remove(key: string): void;
}

class LocalStorageToggleStore implements ToggleStore {
  get(key: string): string | null {
    try {
      return window.localStorage.getItem(key);
    } catch {
      return null;
    }
  }
  set(key: string, value: string): void {
    try {
      window.localStorage.setItem(key, value);
    } catch {
      // localStorage may be unavailable (Safari private mode, quota,
      // sandboxed WebView). Fail silent — toggles reset per session.
    }
  }
  remove(key: string): void {
    try {
      window.localStorage.removeItem(key);
    } catch {
      // see set()
    }
  }
}

export const defaultToggleStore: ToggleStore = new LocalStorageToggleStore();
