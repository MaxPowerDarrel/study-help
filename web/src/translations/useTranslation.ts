import { useEffect, useRef, useState } from "react";

import type { User } from "../auth/api";
import { updateTranslation, type UpdateTranslationResult } from "./api";
import {
  DEFAULT_TRANSLATION,
  isKnownTranslation,
  type TranslationID,
} from "./catalog";
import {
  clearGuestTranslation,
  readGuestTranslation,
  writeGuestTranslation,
} from "./store";

export type TranslationState = {
  // Resolved active translation: from user.translation when signed in,
  // otherwise the guest localStorage value, otherwise the registry
  // default.
  translation: TranslationID;
};

export type TranslationActions = {
  // setTranslation always updates local state and the guest store.
  // When signed in it also persists via PATCH /api/auth/me; for guests
  // it resolves as { kind: "ok-local" } without hitting the network.
  setTranslation: (id: TranslationID) => Promise<UpdateTranslationResult>;
};

// useTranslation is the single source of truth on the client for the
// active translation. It hydrates from the user record when signed in
// and from the guest store (localStorage) otherwise; mutations route
// through PATCH /api/auth/me when signed in and through the guest
// store when not.
//
// Sign-in transition: if the guest's stored pick differs from
// user.translation, the guest pick wins — we PATCH it up to the
// account and clear the guest store on success.
//
// onUserUpdated lets the auth layer keep its cached user in sync when
// a PATCH succeeds.
export function useTranslation(
  user: User | null,
  onUserUpdated?: (user: User) => void,
): TranslationState & TranslationActions {
  const [translation, setLocal] = useState<TranslationID>(() =>
    pickInitial(user),
  );

  const onUserUpdatedRef = useRef(onUserUpdated);
  onUserUpdatedRef.current = onUserUpdated;

  // Track the previous user's identity and stored translation so we
  // can detect (a) sign-in / sign-out transitions and (b) genuine
  // server-side translation changes — without re-running on every new
  // user object reference.
  const prevUserIDRef = useRef<number | null>(user?.id ?? null);
  const prevUserTranslationRef = useRef<string | null>(
    user?.translation ?? null,
  );

  // Latest translation, read by the effect without re-firing on
  // every change.
  const translationRef = useRef(translation);
  translationRef.current = translation;

  useEffect(() => {
    const prevID = prevUserIDRef.current;
    const currID = user?.id ?? null;
    const prevTr = prevUserTranslationRef.current;
    const currTr = user?.translation ?? null;
    prevUserIDRef.current = currID;
    prevUserTranslationRef.current = currTr;

    if (currID !== null && prevID === null) {
      // Sign-in transition. If the guest had picked a translation
      // different from the account's stored value, push it up.
      const guest = readGuestTranslation();
      if (guest && user && guest !== user.translation) {
        setLocal(guest);
        void updateTranslation(guest).then((res) => {
          if (res.kind === "ok") {
            onUserUpdatedRef.current?.(res.user);
            clearGuestTranslation();
          }
        });
        return;
      }
      // No divergent guest pick — clear any stale guest entry and
      // hydrate from the account.
      clearGuestTranslation();
      setLocal(pickInitial(user));
      return;
    }

    if (currID === null && prevID !== null) {
      // Sign-out transition. Persist the last-active translation into
      // the guest store so it survives across the transition and the
      // next reload, instead of snapping back to the registry default.
      writeGuestTranslation(translationRef.current);
      return;
    }

    if (currID !== null && currTr !== prevTr) {
      // Same user, but server-side translation changed (e.g. another
      // tab updated /me). Re-hydrate.
      setLocal(pickInitial(user));
    }
  }, [user]);

  const setTranslation = async (
    id: TranslationID,
  ): Promise<UpdateTranslationResult> => {
    setLocal(id);
    writeGuestTranslation(id);
    if (!user) {
      return { kind: "ok-local" };
    }
    const res = await updateTranslation(id);
    if (res.kind === "ok") {
      onUserUpdatedRef.current?.(res.user);
      clearGuestTranslation();
    } else if (res.kind === "invalid") {
      // Server rejected the choice — revert local state and roll
      // back the guest-store write so it doesn't outlive the user.
      setLocal(pickInitial(user));
      clearGuestTranslation();
    }
    return res;
  };

  return { translation, setTranslation };
}

function pickInitial(user: User | null): TranslationID {
  if (user && isKnownTranslation(user.translation)) {
    return user.translation;
  }
  const guest = readGuestTranslation();
  if (guest) return guest;
  return DEFAULT_TRANSLATION;
}
