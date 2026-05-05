// API client for /api/highlights — same discriminated-union shape as
// src/auth/api.ts so consumers handle errors uniformly.

import type { TranslationID } from "../translations/catalog";

export type Highlight = {
  id: number;
  translation: string;
  book: string;
  chapter: number;
  start_verse: number;
  start_offset: number;
  end_verse: number;
  end_offset: number;
  created_at: string;
};

export type CreateInput = {
  book: string;
  chapter: number;
  start_verse: number;
  start_char_offset: number;
  end_verse: number;
  end_char_offset: number;
};

export type ListResult =
  | { kind: "ok"; highlights: Highlight[] }
  | { kind: "guest" }
  | { kind: "error" };

export type CreateResult =
  | { kind: "ok"; highlight: Highlight }
  | { kind: "overlap" }
  | { kind: "invalid" }
  | { kind: "guest" }
  | { kind: "error" };

export type DeleteResult =
  | { kind: "ok" }
  | { kind: "not_found" }
  | { kind: "guest" }
  | { kind: "error" };

const jsonHeaders = { "Content-Type": "application/json" };

export async function listHighlights(
  book: string,
  chapter: number,
  translation: TranslationID,
): Promise<ListResult> {
  const params = new URLSearchParams({
    book,
    chapter: String(chapter),
    translation,
  });
  let resp: Response;
  try {
    resp = await fetch(`/api/highlights?${params.toString()}`);
  } catch {
    return { kind: "error" };
  }
  if (resp.status === 401) return { kind: "guest" };
  if (!resp.ok) return { kind: "error" };
  try {
    const data = (await resp.json()) as Highlight[];
    return { kind: "ok", highlights: Array.isArray(data) ? data : [] };
  } catch {
    return { kind: "error" };
  }
}

export async function createHighlight(
  input: CreateInput,
  translation: TranslationID,
): Promise<CreateResult> {
  let resp: Response;
  try {
    resp = await fetch("/api/highlights", {
      method: "POST",
      headers: jsonHeaders,
      body: JSON.stringify({ ...input, translation }),
    });
  } catch {
    return { kind: "error" };
  }
  if (resp.status === 401) return { kind: "guest" };
  if (resp.status === 409) return { kind: "overlap" };
  if (resp.status === 400) return { kind: "invalid" };
  if (!resp.ok) return { kind: "error" };
  try {
    const highlight = (await resp.json()) as Highlight;
    return { kind: "ok", highlight };
  } catch {
    return { kind: "error" };
  }
}

export async function deleteHighlight(id: number): Promise<DeleteResult> {
  let resp: Response;
  try {
    resp = await fetch(`/api/highlights/${id}`, { method: "DELETE" });
  } catch {
    return { kind: "error" };
  }
  if (resp.status === 401) return { kind: "guest" };
  if (resp.status === 404) return { kind: "not_found" };
  if (!resp.ok) return { kind: "error" };
  return { kind: "ok" };
}
