import { useState } from "react";
import { DEFAULT_TRANSLATION, type TranslationID } from "./catalog";
import { readStoredTranslation, writeStoredTranslation } from "./store";

export type TranslationState = {
  translation: TranslationID;
  setTranslation: (id: TranslationID) => void;
};

// useTranslation owns the active translation choice. It hydrates from
// localStorage and writes back on every change; with no auth in v2,
// this is the single source of truth on the client.
export function useTranslation(): TranslationState {
  const [translation, setLocal] = useState<TranslationID>(
    () => readStoredTranslation() ?? DEFAULT_TRANSLATION,
  );

  const setTranslation = (id: TranslationID) => {
    setLocal(id);
    writeStoredTranslation(id);
  };

  return { translation, setTranslation };
}
