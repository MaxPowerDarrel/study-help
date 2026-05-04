import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AuthChip } from "./AuthChip";
import type { User } from "./api";

const sampleUser: User = {
  id: 1,
  email: "alice@example.com",
  created_at: "2026-05-04T00:00:00Z",
  updated_at: "2026-05-04T00:00:00Z",
};

describe("AuthChip", () => {
  it("guest: renders Sign in button and calls onOpenSignin on click", async () => {
    const user = userEvent.setup();
    const onOpenSignin = vi.fn();
    render(
      <AuthChip user={null} onOpenSignin={onOpenSignin} onSignout={vi.fn()} />,
    );
    await user.click(screen.getByRole("button", { name: /sign in/i }));
    expect(onOpenSignin).toHaveBeenCalledTimes(1);
  });

  it("signed-in: renders email chip and reveals menu on click", async () => {
    const user = userEvent.setup();
    render(
      <AuthChip user={sampleUser} onOpenSignin={vi.fn()} onSignout={vi.fn()} />,
    );
    // Email is visible in the chip itself.
    expect(screen.getAllByText(/alice@example\.com/i).length).toBeGreaterThan(
      0,
    );
    // Menu is initially closed.
    expect(screen.queryByRole("menuitem")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /alice/i }));
    expect(
      screen.getByRole("menuitem", { name: /sign out/i }),
    ).toBeInTheDocument();
  });

  it("signed-in: calls onSignout when sign-out is clicked", async () => {
    const user = userEvent.setup();
    const onSignout = vi.fn();
    render(
      <AuthChip
        user={sampleUser}
        onOpenSignin={vi.fn()}
        onSignout={onSignout}
      />,
    );
    await user.click(screen.getByRole("button", { name: /alice/i }));
    await user.click(screen.getByRole("menuitem", { name: /sign out/i }));
    expect(onSignout).toHaveBeenCalledTimes(1);
    // Menu closes after click.
    expect(screen.queryByRole("menuitem")).not.toBeInTheDocument();
  });
});
