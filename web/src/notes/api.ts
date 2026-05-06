// API client for /api/notes — discriminated-union shape matching
// src/highlights/api.ts so consumers handle errors uniformly.

import type { TranslationID } from "../translations/catalog";

export type Note = {
  id: number;
  translation: string;
  book: string;
  chapter: number;
  start_verse: number;
  start_offset: number;
  end_verse: number;
  end_offset: number;
  body: string;
  created_at: string;
  updated_at: string;
};

export type CreateInput = {
  book: string;
  chapter: number;
  start_verse: number;
  start_offset: number;
  end_verse: number;
  end_offset: number;
  body: string;
};

export type ListResult =
  | { kind: "ok"; notes: Note[] }
  | { kind: "guest" }
  | { kind: "error" };

export type CreateResult =
  | { kind: "ok"; note: Note }
  | { kind: "invalid" }
  | { kind: "guest" }
  | { kind: "error" };

export type UpdateResult =
  | { kind: "ok"; note: Note }
  | { kind: "invalid" }
  | { kind: "not_found" }
  | { kind: "guest" }
  | { kind: "error" };

export type DeleteResult =
  | { kind: "ok" }
  | { kind: "not_found" }
  | { kind: "guest" }
  | { kind: "error" };

const jsonHeaders = { "Content-Type": "application/json" };

export async function listNotes(
  book: string,
  chapter: number,
  translation: TranslationID,
  signal?: AbortSignal,
): Promise<ListResult> {
  const params = new URLSearchParams({
    book,
    chapter: String(chapter),
    translation,
  });
  let resp: Response;
  try {
    resp = await fetch(`/api/notes?${params.toString()}`, { signal });
  } catch {
    return { kind: "error" };
  }
  if (resp.status === 401) return { kind: "guest" };
  if (!resp.ok) return { kind: "error" };
  try {
    const data = (await resp.json()) as Note[];
    return { kind: "ok", notes: Array.isArray(data) ? data : [] };
  } catch {
    return { kind: "error" };
  }
}

export async function createNote(
  input: CreateInput,
  translation: TranslationID,
): Promise<CreateResult> {
  let resp: Response;
  try {
    resp = await fetch("/api/notes", {
      method: "POST",
      headers: jsonHeaders,
      body: JSON.stringify({ ...input, translation }),
    });
  } catch {
    return { kind: "error" };
  }
  if (resp.status === 401) return { kind: "guest" };
  if (resp.status === 400) return { kind: "invalid" };
  if (!resp.ok) return { kind: "error" };
  try {
    const note = (await resp.json()) as Note;
    return { kind: "ok", note };
  } catch {
    return { kind: "error" };
  }
}

export async function updateNote(
  id: number,
  body: string,
): Promise<UpdateResult> {
  let resp: Response;
  try {
    resp = await fetch(`/api/notes/${id}`, {
      method: "PATCH",
      headers: jsonHeaders,
      body: JSON.stringify({ body }),
    });
  } catch {
    return { kind: "error" };
  }
  if (resp.status === 401) return { kind: "guest" };
  if (resp.status === 404) return { kind: "not_found" };
  if (resp.status === 400) return { kind: "invalid" };
  if (!resp.ok) return { kind: "error" };
  try {
    const note = (await resp.json()) as Note;
    return { kind: "ok", note };
  } catch {
    return { kind: "error" };
  }
}

export async function deleteNote(id: number): Promise<DeleteResult> {
  let resp: Response;
  try {
    resp = await fetch(`/api/notes/${id}`, { method: "DELETE" });
  } catch {
    return { kind: "error" };
  }
  if (resp.status === 401) return { kind: "guest" };
  if (resp.status === 404) return { kind: "not_found" };
  if (!resp.ok) return { kind: "error" };
  return { kind: "ok" };
}
