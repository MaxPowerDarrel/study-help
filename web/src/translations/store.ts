// Guest translation persistence. When a user is signed out, the active
// translation is stored under this key in defaultToggleStore (the §4
// platform abstraction over localStorage). On sign-in, the value is
// pushed to the account via PATCH /api/auth/me and cleared.
//
// Mirrors the shape of web/src/theme.ts.

import { defaultToggleStore, ToggleStore } from "../platform/ToggleStore";
import { isKnownTranslation, type TranslationID } from "./catalog";

const KEY = "translation";

export function readGuestTranslation(
  store: ToggleStore = defaultToggleStore,
): TranslationID | null {
  const v = store.get(KEY);
  return v && isKnownTranslation(v) ? v : null;
}

export function writeGuestTranslation(
  id: TranslationID,
  store: ToggleStore = defaultToggleStore,
): void {
  store.set(KEY, id);
}

export function clearGuestTranslation(
  store: ToggleStore = defaultToggleStore,
): void {
  store.remove(KEY);
}
