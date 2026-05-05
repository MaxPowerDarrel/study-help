import { useEffect, useState } from "react";

import type { User } from "../auth/api";
import { updateTranslation, type UpdateTranslationResult } from "./api";
import {
  DEFAULT_TRANSLATION,
  isKnownTranslation,
  type TranslationID,
} from "./catalog";

export type TranslationState = {
  // Resolved active translation: from user.translation when signed in,
  // otherwise the registry default.
  translation: TranslationID;
};

export type TranslationActions = {
  // setTranslation persists the choice via PATCH /api/auth/me when the
  // user is signed in. Guests get { kind: "guest" } — the picker is
  // gated upstream so this branch is defensive.
  setTranslation: (id: TranslationID) => Promise<UpdateTranslationResult>;
};

// useTranslation is the single source of truth on the client for the
// active translation. It hydrates from the user record (no localStorage
// fallback) and routes mutations through PATCH /api/auth/me.
//
// onUserUpdated lets the auth layer keep its cached user in sync when a
// PATCH succeeds — without it, useUser's local copy would lag.
export function useTranslation(
  user: User | null,
  onUserUpdated?: (user: User) => void,
): TranslationState & TranslationActions {
  const [translation, setLocal] = useState<TranslationID>(() =>
    pickInitial(user),
  );

  useEffect(() => {
    setLocal(pickInitial(user));
  }, [user]);

  const setTranslation = async (
    id: TranslationID,
  ): Promise<UpdateTranslationResult> => {
    const res = await updateTranslation(id);
    if (res.kind === "ok") {
      setLocal(id);
      onUserUpdated?.(res.user);
    }
    return res;
  };

  return { translation, setTranslation };
}

function pickInitial(user: User | null): TranslationID {
  if (user && isKnownTranslation(user.translation)) {
    return user.translation;
  }
  return DEFAULT_TRANSLATION;
}
