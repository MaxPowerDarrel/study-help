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
  // subscribe registers a listener fired on any set/remove against this
  // store, plus cross-tab `storage` events for the LocalStorage-backed
  // implementation. Returns an unsubscribe function. Used by
  // useSyncExternalStore consumers.
  subscribe(listener: () => void): () => void;
}

class LocalStorageToggleStore implements ToggleStore {
  private listeners = new Set<() => void>();
  private windowListener: ((e: StorageEvent) => void) | null = null;

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
    this.fire();
  }
  remove(key: string): void {
    try {
      window.localStorage.removeItem(key);
    } catch {
      // see set()
    }
    this.fire();
  }
  subscribe(listener: () => void): () => void {
    this.listeners.add(listener);
    if (!this.windowListener && typeof window !== "undefined") {
      // Cross-tab updates: localStorage fires `storage` on other tabs
      // (not the originating one), so writes from the same tab are
      // covered by fire() above.
      this.windowListener = () => this.fire();
      window.addEventListener("storage", this.windowListener);
    }
    return () => {
      this.listeners.delete(listener);
      if (this.listeners.size === 0 && this.windowListener) {
        window.removeEventListener("storage", this.windowListener);
        this.windowListener = null;
      }
    };
  }
  private fire(): void {
    for (const l of this.listeners) l();
  }
}

export const defaultToggleStore: ToggleStore = new LocalStorageToggleStore();
