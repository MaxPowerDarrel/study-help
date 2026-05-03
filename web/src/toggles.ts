import { useEffect, useState } from "react";
import { defaultToggleStore, ToggleStore } from "./platform/ToggleStore";

export type Toggles = {
  include_headings: boolean;
  include_footnotes: boolean;
  include_verse_numbers: boolean;
  include_passage_references: boolean;
  include_word_of_christ: boolean;
};

export const DEFAULT_TOGGLES: Toggles = {
  include_headings: true,
  include_footnotes: true,
  include_verse_numbers: true,
  include_passage_references: true,
  include_word_of_christ: true,
};

const STORAGE_KEY = "reader.toggles";

export function useToggles(store: ToggleStore = defaultToggleStore) {
  const [toggles, setToggles] = useState<Toggles>(() => {
    const raw = store.get(STORAGE_KEY);
    if (!raw) return DEFAULT_TOGGLES;
    try {
      return { ...DEFAULT_TOGGLES, ...JSON.parse(raw) };
    } catch {
      return DEFAULT_TOGGLES;
    }
  });

  useEffect(() => {
    store.set(STORAGE_KEY, JSON.stringify(toggles));
  }, [toggles, store]);

  return [toggles, setToggles] as const;
}

export function togglesToQuery(t: Toggles): string {
  const params = new URLSearchParams();
  params.set("include_headings", String(t.include_headings));
  params.set("include_footnotes", String(t.include_footnotes));
  params.set("include_verse_numbers", String(t.include_verse_numbers));
  params.set("include_passage_references", String(t.include_passage_references));
  params.set("include_word_of_christ", String(t.include_word_of_christ));
  return params.toString();
}
