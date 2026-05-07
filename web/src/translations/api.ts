// API client for PATCH /api/auth/me. Same discriminated-union posture
// as src/auth/api.ts.

import type { User } from "../auth/api";
import type { TranslationID } from "./catalog";

export type UpdateTranslationResult =
  | { kind: "ok"; user: User }
  | { kind: "ok-local" }
  | { kind: "guest" }
  | { kind: "invalid" }
  | { kind: "error" };

export async function updateTranslation(
  id: TranslationID,
): Promise<UpdateTranslationResult> {
  let resp: Response;
  try {
    resp = await fetch("/api/auth/me", {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ translation: id }),
    });
  } catch {
    return { kind: "error" };
  }
  if (resp.status === 401) return { kind: "guest" };
  if (resp.status === 400) return { kind: "invalid" };
  if (!resp.ok) return { kind: "error" };
  try {
    const user = (await resp.json()) as User;
    return { kind: "ok", user };
  } catch {
    return { kind: "error" };
  }
}
