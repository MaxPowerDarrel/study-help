// Translation persistence in localStorage. Read on first render,
// updated on every picker change. Mirrors the shape of web/src/theme.ts.

import { defaultToggleStore, ToggleStore } from "../platform/ToggleStore";
import { isKnownTranslation, type TranslationID } from "./catalog";

const KEY = "translation";

export function readStoredTranslation(
  store: ToggleStore = defaultToggleStore,
): TranslationID | null {
  const v = store.get(KEY);
  return v && isKnownTranslation(v) ? v : null;
}

export function writeStoredTranslation(
  id: TranslationID,
  store: ToggleStore = defaultToggleStore,
): void {
  store.set(KEY, id);
}
