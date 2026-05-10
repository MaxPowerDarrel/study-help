// restore.ts owns "where the user was" — three small localStorage-backed
// stores (active tab, read-tab passage, daily-tab date) so reopening the
// app feels like resuming, not restarting. See specs/restore-last-location.md.
//
// Hooks follow the useState + write-on-set pattern from useTranslation,
// not useSyncExternalStore: cross-tab live sync of "where you were" would
// be disorienting (a tab change in window A shouldn't yank window B
// mid-reading).

import { useState } from "react";
import { CANON, type ChapterRef } from "./canon";
import { defaultToggleStore, type ToggleStore } from "./platform/ToggleStore";

export type Tab = "read" | "daily";

const TAB_KEY = "reader.active-tab";
const READ_REF_KEY = "reader.read-location";
const DAILY_DATE_KEY = "reader.daily-date";

const DEFAULT_TAB: Tab = "read";
const DEFAULT_READ_REF: ChapterRef = { bookIndex: 42, chapter: 3 }; // John 3

// --- Active tab ---

export function readStoredTab(
  store: ToggleStore = defaultToggleStore,
): Tab | null {
  const v = store.get(TAB_KEY);
  return v === "read" || v === "daily" ? v : null;
}

export function writeStoredTab(
  tab: Tab,
  store: ToggleStore = defaultToggleStore,
): void {
  store.set(TAB_KEY, tab);
}

export function useStoredTab(
  store: ToggleStore = defaultToggleStore,
): [Tab, (t: Tab) => void] {
  const [tab, setLocal] = useState<Tab>(
    () => readStoredTab(store) ?? DEFAULT_TAB,
  );
  const setTab = (t: Tab) => {
    setLocal(t);
    writeStoredTab(t, store);
  };
  return [tab, setTab];
}

// --- Read-tab passage ---

export function readStoredReadRef(
  store: ToggleStore = defaultToggleStore,
): ChapterRef | null {
  const raw = store.get(READ_REF_KEY);
  if (!raw) return null;
  try {
    const parsed: unknown = JSON.parse(raw);
    if (
      !parsed ||
      typeof parsed !== "object" ||
      typeof (parsed as { bookIndex?: unknown }).bookIndex !== "number" ||
      typeof (parsed as { chapter?: unknown }).chapter !== "number"
    ) {
      return null;
    }
    const { bookIndex, chapter } = parsed as ChapterRef;
    if (
      !Number.isInteger(bookIndex) ||
      bookIndex < 0 ||
      bookIndex >= CANON.length
    ) {
      return null;
    }
    const book = CANON[bookIndex];
    if (!Number.isInteger(chapter) || chapter < 1 || chapter > book.chapters) {
      return null;
    }
    return { bookIndex, chapter };
  } catch {
    return null;
  }
}

export function writeStoredReadRef(
  ref: ChapterRef,
  store: ToggleStore = defaultToggleStore,
): void {
  store.set(READ_REF_KEY, JSON.stringify(ref));
}

export function useStoredReadRef(
  store: ToggleStore = defaultToggleStore,
): [ChapterRef, (r: ChapterRef) => void] {
  const [ref, setLocal] = useState<ChapterRef>(
    () => readStoredReadRef(store) ?? DEFAULT_READ_REF,
  );
  const setRef = (r: ChapterRef) => {
    setLocal(r);
    writeStoredReadRef(r, store);
  };
  return [ref, setRef];
}

// --- Daily-tab date (with snap-to-today) ---

// The stored entry pairs the selected `date` with the `savedOn` calendar
// day. On read, if savedOn !== today, the stored value is treated as
// expired and the hook returns today instead — pinning a non-today date
// is meaningful within a session, but stale across days.
type StoredDailyDate = { date: string; savedOn: string };

export function readStoredDailyDate(
  today: string,
  store: ToggleStore = defaultToggleStore,
): string | null {
  const raw = store.get(DAILY_DATE_KEY);
  if (!raw) return null;
  try {
    const parsed: unknown = JSON.parse(raw);
    if (
      !parsed ||
      typeof parsed !== "object" ||
      typeof (parsed as { date?: unknown }).date !== "string" ||
      typeof (parsed as { savedOn?: unknown }).savedOn !== "string"
    ) {
      return null;
    }
    const { date, savedOn } = parsed as StoredDailyDate;
    if (savedOn !== today) return null;
    return date;
  } catch {
    return null;
  }
}

export function writeStoredDailyDate(
  date: string,
  today: string,
  store: ToggleStore = defaultToggleStore,
): void {
  const entry: StoredDailyDate = { date, savedOn: today };
  store.set(DAILY_DATE_KEY, JSON.stringify(entry));
}

export function useStoredDailyDate(
  today: string,
  store: ToggleStore = defaultToggleStore,
): [string, (d: string) => void] {
  const [date, setLocal] = useState<string>(
    () => readStoredDailyDate(today, store) ?? today,
  );
  const setDate = (d: string) => {
    setLocal(d);
    writeStoredDailyDate(d, today, store);
  };
  return [date, setDate];
}
