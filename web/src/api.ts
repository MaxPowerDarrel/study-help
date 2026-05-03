import { Toggles, togglesToQuery } from "./toggles";

export type FetchResult =
  | { kind: "ok"; html: string; canonical: string }
  | { kind: "rate_limited" }
  | { kind: "error" };

export type DailyPassage = {
  book: string;
  chapters: string;
  testament: "OT" | "NT";
};

export type DailyResult =
  | { kind: "ok"; passages: DailyPassage[] }
  | { kind: "empty" }
  | { kind: "error" };

type EsvJson = { canonical?: string; passages?: string[] };

type DailyJson = { passages?: DailyPassage[]; message?: string };

export async function fetchPassage(
  q: string,
  toggles: Toggles,
): Promise<FetchResult> {
  const params = new URLSearchParams(togglesToQuery(toggles));
  params.set("q", q);
  let resp: Response;
  try {
    resp = await fetch(`/api/passage?${params.toString()}`);
  } catch {
    return { kind: "error" };
  }
  if (resp.status === 429) return { kind: "rate_limited" };
  if (!resp.ok) return { kind: "error" };
  let data: EsvJson;
  try {
    data = await resp.json();
  } catch {
    return { kind: "error" };
  }
  const html = (data.passages ?? []).join("");
  return { kind: "ok", html, canonical: data.canonical ?? q };
}

export async function fetchDailyReading(tz: string): Promise<DailyResult> {
  let resp: Response;
  try {
    resp = await fetch(`/api/daily-reading?tz=${encodeURIComponent(tz)}`);
  } catch {
    return { kind: "error" };
  }
  if (!resp.ok) return { kind: "error" };
  let data: DailyJson;
  try {
    data = await resp.json();
  } catch {
    return { kind: "error" };
  }
  const passages = data.passages ?? [];
  if (passages.length === 0) return { kind: "empty" };
  return { kind: "ok", passages };
}
