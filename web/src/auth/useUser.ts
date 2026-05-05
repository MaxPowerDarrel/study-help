import { useEffect, useState } from "react";
import {
  me,
  signin as apiSignin,
  signup as apiSignup,
  signout as apiSignout,
  type SigninResult,
  type SignupResult,
  type User,
} from "./api";

export type UserState = {
  user: User | null;
  hydrating: boolean; // true until the cold-load /me completes
};

export type UserActions = {
  signin: (email: string, password: string) => Promise<SigninResult>;
  signup: (email: string, password: string) => Promise<SignupResult>;
  signout: () => Promise<void>;
  // applyUser replaces the cached user. Used by PATCH /api/auth/me
  // callers (e.g., useTranslation) to keep this hook's copy in sync.
  applyUser: (user: User) => void;
};

// useUser owns the current-user state and the auth-mutating actions.
// On mount it hydrates from GET /api/auth/me; the chip and the panel
// both consume what this hook returns.
export function useUser(): UserState & UserActions {
  const [user, setUser] = useState<User | null>(null);
  const [hydrating, setHydrating] = useState(true);

  useEffect(() => {
    let cancelled = false;
    me().then((res) => {
      if (cancelled) return;
      if (res.kind === "ok") setUser(res.user);
      setHydrating(false);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  const signin = async (
    email: string,
    password: string,
  ): Promise<SigninResult> => {
    const res = await apiSignin(email, password);
    if (res.kind === "ok") setUser(res.user);
    return res;
  };

  const signup = async (
    email: string,
    password: string,
  ): Promise<SignupResult> => {
    const res = await apiSignup(email, password);
    if (res.kind === "ok") setUser(res.user);
    return res;
  };

  const signout = async (): Promise<void> => {
    await apiSignout();
    setUser(null);
  };

  const applyUser = (u: User): void => {
    setUser(u);
  };

  return { user, hydrating, signin, signup, signout, applyUser };
}
