import { Toggles, togglesToQuery } from "./toggles";

export type FetchResult =
  | { kind: "ok"; html: string; canonical: string }
  | { kind: "rate_limited" }
  | { kind: "error" };

type EsvJson = { canonical?: string; passages?: string[] };

export async function fetchPassage(q: string, toggles: Toggles): Promise<FetchResult> {
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
