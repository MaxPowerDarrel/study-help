import { useState } from "react";
import styles from "./AuthPanel.module.css";
import type { SigninResult, SignupResult } from "./api";

const MIN_PASSWORD_LEN = 12;

type Mode = "signin" | "signup";

type Props = {
  open: boolean;
  onClose: () => void;
  signin: (email: string, password: string) => Promise<SigninResult>;
  signup: (email: string, password: string) => Promise<SignupResult>;
};

export function AuthPanel({ open, onClose, signin, signup }: Props) {
  const [mode, setMode] = useState<Mode>("signin");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  if (!open) return null;

  const reset = () => {
    setEmail("");
    setPassword("");
    setError(null);
    setSubmitting(false);
  };

  const handleClose = () => {
    reset();
    onClose();
  };

  const flipMode = () => {
    setMode(mode === "signin" ? "signup" : "signin");
    setError(null);
  };

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    if (!email.trim()) {
      setError("Email is required");
      return;
    }
    if (mode === "signup" && password.length < MIN_PASSWORD_LEN) {
      setError(`Password must be at least ${MIN_PASSWORD_LEN} characters`);
      return;
    }
    if (!password) {
      setError("Password is required");
      return;
    }
    setSubmitting(true);
    const res =
      mode === "signin"
        ? await signin(email, password)
        : await signup(email, password);
    setSubmitting(false);
    if (res.kind === "ok") {
      reset();
      onClose();
      return;
    }
    setError(messageFor(res.kind));
  };

  return (
    <div className={styles.overlay} role="dialog" aria-label="Account">
      <div className={styles.pane}>
        <header className={styles.header}>
          <h2>{mode === "signin" ? "Sign in" : "Create account"}</h2>
          <button
            type="button"
            className={styles.close}
            onClick={handleClose}
            aria-label="Close"
          >
            ×
          </button>
        </header>

        <form className={styles.form} onSubmit={onSubmit} noValidate>
          <div className={styles.field}>
            <label htmlFor="auth-email">Email</label>
            <input
              id="auth-email"
              type="email"
              autoComplete="email"
              autoCapitalize="none"
              autoCorrect="off"
              spellCheck={false}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>

          <div className={styles.field}>
            <label htmlFor="auth-password">Password</label>
            <input
              id="auth-password"
              type="password"
              autoComplete={
                mode === "signin" ? "current-password" : "new-password"
              }
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
            {mode === "signup" && (
              <span className={styles.hint}>
                At least {MIN_PASSWORD_LEN} characters.
              </span>
            )}
          </div>

          <p
            className={styles.error}
            role="alert"
            aria-live="polite"
            data-testid="auth-error"
          >
            {error ?? ""}
          </p>

          <button type="submit" className={styles.submit} disabled={submitting}>
            {submitting
              ? "Working…"
              : mode === "signin"
                ? "Sign in"
                : "Create account"}
          </button>
        </form>

        <p className={styles.modeSwitch}>
          {mode === "signin" ? (
            <>
              No account?{" "}
              <button type="button" onClick={flipMode}>
                Create one
              </button>
            </>
          ) : (
            <>
              Already have an account?{" "}
              <button type="button" onClick={flipMode}>
                Sign in
              </button>
            </>
          )}
        </p>
      </div>
    </div>
  );
}

function messageFor(kind: string): string {
  switch (kind) {
    case "invalid_credentials":
      return "Invalid email or password";
    case "weak_password":
      return `Password must be at least ${MIN_PASSWORD_LEN} characters`;
    case "email_taken":
      return "An account with that email already exists";
    case "rate_limited":
      return "Too many attempts. Try again in a few minutes.";
    default:
      return "Something went wrong. Try again.";
  }
}
