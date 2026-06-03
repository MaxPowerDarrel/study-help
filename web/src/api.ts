import { type PlanID } from "./daily/plans";
import { Toggles, togglesToQuery } from "./toggles";
import type { TranslationID } from "./translations/catalog";

export type FetchResult =
  | { kind: "ok"; html: string; canonical: string }
  | { kind: "rate_limited" }
  | { kind: "error" };

export type DailyPassage = {
  book: string;
  chapters: string;
  testament: "OT" | "NT" | "";
};

export type DailyPlanResult = {
  id: PlanID;
  name: string;
  passages: DailyPassage[];
  message: string;
};

export type DailyResult =
  | { kind: "ok"; plans: DailyPlanResult[] }
  | { kind: "error" };

type EsvJson = { canonical?: string; passages?: string[] };

type DailyJson = {
  plans?: {
    id?: string;
    name?: string;
    passages?: DailyPassage[];
    message?: string;
  }[];
};

export async function fetchPassage(
  q: string,
  toggles: Toggles,
  translation: TranslationID,
): Promise<FetchResult> {
  const params = new URLSearchParams(togglesToQuery(toggles));
  params.set("q", q);
  params.set("translation", translation);
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

// fetchCrossref looks up the verse text behind a cross-reference marker
// for the popover. `q` is the reference string lifted from the marker's
// href (e.g. "Job 38:4-7; Psalm 33:6"). The endpoint is ESV-only — see
// internal/server/crossref.go — and returns the same {passages} envelope
// as /api/passage, so the result renders with the shared .passage styles.
export async function fetchCrossref(q: string): Promise<FetchResult> {
  const params = new URLSearchParams({ q });
  let resp: Response;
  try {
    resp = await fetch(`/api/crossref?${params.toString()}`);
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

export async function fetchDailyReading(
  tz: string,
  planIDs: PlanID[],
  date?: string,
): Promise<DailyResult> {
  const params = new URLSearchParams({ tz });
  if (date) params.set("date", date);
  if (planIDs.length > 0) params.set("plans", planIDs.join(","));
  let resp: Response;
  try {
    resp = await fetch(`/api/daily-reading?${params.toString()}`);
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
  const plans: DailyPlanResult[] = (data.plans ?? []).map((p) => ({
    id: (p.id ?? "") as PlanID,
    name: p.name ?? "",
    passages: p.passages ?? [],
    message: p.message ?? "",
  }));
  return { kind: "ok", plans };
}
