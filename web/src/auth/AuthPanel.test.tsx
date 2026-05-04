import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AuthPanel } from "./AuthPanel";
import type { SigninResult, SignupResult, User } from "./api";

const sampleUser: User = {
  id: 1,
  email: "alice@example.com",
  created_at: "2026-05-04T00:00:00Z",
  updated_at: "2026-05-04T00:00:00Z",
};

function ok<T extends SigninResult | SignupResult>(): T {
  return { kind: "ok", user: sampleUser } as T;
}

describe("AuthPanel", () => {
  it("returns null when closed", () => {
    const { container } = render(
      <AuthPanel
        open={false}
        onClose={vi.fn()}
        signin={vi.fn()}
        signup={vi.fn()}
      />,
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("submits signin and closes on success", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const signin = vi.fn().mockResolvedValue(ok<SigninResult>());
    render(
      <AuthPanel open onClose={onClose} signin={signin} signup={vi.fn()} />,
    );
    await user.type(screen.getByLabelText(/email/i), "alice@example.com");
    await user.type(screen.getByLabelText(/password/i), "twelvecharspw");
    await user.click(screen.getByRole("button", { name: /^sign in$/i }));
    expect(signin).toHaveBeenCalledWith("alice@example.com", "twelvecharspw");
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("shows ambiguous error on invalid credentials", async () => {
    const user = userEvent.setup();
    const signin = vi.fn(
      async (_e: string, _p: string): Promise<SigninResult> => ({
        kind: "invalid_credentials",
      }),
    );
    render(
      <AuthPanel open onClose={vi.fn()} signin={signin} signup={vi.fn()} />,
    );
    await user.type(screen.getByLabelText(/email/i), "alice@example.com");
    await user.type(screen.getByLabelText(/password/i), "twelvecharspw");
    await user.click(screen.getByRole("button", { name: /^sign in$/i }));
    expect(screen.getByTestId("auth-error")).toHaveTextContent(
      /invalid email or password/i,
    );
  });

  it("toggles to sign-up mode and validates 12-char password client-side", async () => {
    const user = userEvent.setup();
    const signup = vi.fn();
    render(
      <AuthPanel open onClose={vi.fn()} signin={vi.fn()} signup={signup} />,
    );
    await user.click(screen.getByRole("button", { name: /create one/i }));
    // Heading flipped to "Create account".
    expect(
      screen.getByRole("heading", { name: /create account/i }),
    ).toBeInTheDocument();
    // Submit with a short password — server must NOT be called.
    await user.type(screen.getByLabelText(/email/i), "alice@example.com");
    await user.type(screen.getByLabelText(/password/i), "short");
    await user.click(screen.getByRole("button", { name: /^create account$/i }));
    expect(signup).not.toHaveBeenCalled();
    expect(screen.getByTestId("auth-error")).toHaveTextContent(/12 characters/);
  });

  it("signup: shows email-taken error", async () => {
    const user = userEvent.setup();
    const signup = vi.fn(
      async (_e: string, _p: string): Promise<SignupResult> => ({
        kind: "email_taken",
      }),
    );
    render(
      <AuthPanel open onClose={vi.fn()} signin={vi.fn()} signup={signup} />,
    );
    await user.click(screen.getByRole("button", { name: /create one/i }));
    await user.type(screen.getByLabelText(/email/i), "alice@example.com");
    await user.type(screen.getByLabelText(/password/i), "twelvecharspw");
    await user.click(screen.getByRole("button", { name: /^create account$/i }));
    expect(screen.getByTestId("auth-error")).toHaveTextContent(/already/i);
  });
});
