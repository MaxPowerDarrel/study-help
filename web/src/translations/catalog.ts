// Client-side mirror of the server's scripture.Registry. When a new
// translation is registered server-side (internal/scripture, internal/<id>),
// add a matching entry here in the same diff.

export type TranslationID = "ESV";

export type TranslationEntry = {
  id: TranslationID;
  displayName: string;
};

export const TRANSLATIONS: ReadonlyArray<TranslationEntry> = [
  { id: "ESV", displayName: "English Standard Version" },
];

export const DEFAULT_TRANSLATION: TranslationID = "ESV";

export function isKnownTranslation(s: string): s is TranslationID {
  return TRANSLATIONS.some((t) => t.id === s);
}
