// API client for the /api/auth/* endpoints. Same discriminated-union
// shape as src/api.ts so consumers handle errors uniformly.

export type User = {
  id: number;
  email: string;
  translation: string;
  created_at: string;
  updated_at: string;
};

export type SigninResult =
  | { kind: "ok"; user: User }
  | { kind: "invalid_credentials" }
  | { kind: "rate_limited" }
  | { kind: "error" };

export type SignupResult =
  | { kind: "ok"; user: User }
  | { kind: "weak_password" }
  | { kind: "email_taken" }
  | { kind: "rate_limited" }
  | { kind: "error" };

export type MeResult =
  | { kind: "ok"; user: User }
  | { kind: "guest" }
  | { kind: "error" };

const jsonHeaders = { "Content-Type": "application/json" };

async function readUser(resp: Response): Promise<User | null> {
  try {
    return (await resp.json()) as User;
  } catch {
    return null;
  }
}

export async function signin(
  email: string,
  password: string,
): Promise<SigninResult> {
  let resp: Response;
  try {
    resp = await fetch("/api/auth/signin", {
      method: "POST",
      headers: jsonHeaders,
      body: JSON.stringify({ email, password }),
    });
  } catch {
    return { kind: "error" };
  }
  if (resp.status === 401) return { kind: "invalid_credentials" };
  if (resp.status === 429) return { kind: "rate_limited" };
  if (!resp.ok) return { kind: "error" };
  const user = await readUser(resp);
  return user ? { kind: "ok", user } : { kind: "error" };
}

export async function signup(
  email: string,
  password: string,
): Promise<SignupResult> {
  let resp: Response;
  try {
    resp = await fetch("/api/auth/signup", {
      method: "POST",
      headers: jsonHeaders,
      body: JSON.stringify({ email, password }),
    });
  } catch {
    return { kind: "error" };
  }
  if (resp.status === 400) return { kind: "weak_password" };
  if (resp.status === 409) return { kind: "email_taken" };
  if (resp.status === 429) return { kind: "rate_limited" };
  if (!resp.ok) return { kind: "error" };
  const user = await readUser(resp);
  return user ? { kind: "ok", user } : { kind: "error" };
}

// signout always succeeds (server returns 204 idempotently). Network
// errors are swallowed; the client clears local state regardless.
export async function signout(): Promise<void> {
  try {
    await fetch("/api/auth/signout", { method: "POST" });
  } catch {
    // ignore
  }
}

export async function me(): Promise<MeResult> {
  let resp: Response;
  try {
    resp = await fetch("/api/auth/me");
  } catch {
    return { kind: "error" };
  }
  if (resp.status === 401) return { kind: "guest" };
  if (!resp.ok) return { kind: "error" };
  const user = await readUser(resp);
  return user ? { kind: "ok", user } : { kind: "error" };
}
