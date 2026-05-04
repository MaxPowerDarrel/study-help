import { useEffect, useRef, useState } from "react";
import styles from "./AuthChip.module.css";
import type { User } from "./api";

type Props = {
  user: User | null;
  onOpenSignin: () => void;
  onSignout: () => void;
};

// AuthChip lives in the top-right of the header. Guest: a "Sign in"
// button that opens the auth panel. Signed-in: an email chip that
// reveals a small menu with "Sign out". Click-outside closes the menu.
export function AuthChip({ user, onOpenSignin, onSignout }: Props) {
  const [menuOpen, setMenuOpen] = useState(false);
  const wrapperRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!menuOpen) return;
    const onDocClick = (e: MouseEvent) => {
      if (
        wrapperRef.current &&
        !wrapperRef.current.contains(e.target as Node)
      ) {
        setMenuOpen(false);
      }
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setMenuOpen(false);
    };
    document.addEventListener("mousedown", onDocClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDocClick);
      document.removeEventListener("keydown", onKey);
    };
  }, [menuOpen]);

  if (!user) {
    return (
      <button type="button" className={styles.signin} onClick={onOpenSignin}>
        Sign in
      </button>
    );
  }

  return (
    <div className={styles.wrapper} ref={wrapperRef}>
      <button
        type="button"
        className={styles.account}
        aria-haspopup="menu"
        aria-expanded={menuOpen}
        onClick={() => setMenuOpen((v) => !v)}
      >
        <span className={styles.email}>{user.email}</span>
        <span className={styles.caret} aria-hidden="true">
          ▾
        </span>
      </button>
      {menuOpen && (
        <div className={styles.menu} role="menu">
          <div className={styles.menuEmail}>{user.email}</div>
          <button
            type="button"
            role="menuitem"
            className={styles.menuButton}
            onClick={() => {
              setMenuOpen(false);
              onSignout();
            }}
          >
            Sign out
          </button>
        </div>
      )}
    </div>
  );
}
