import { describe, expect, it, vi, beforeEach, type Mock } from "vitest";
import {
  render,
  screen,
  fireEvent,
  waitFor,
  cleanup,
} from "@testing-library/react";
import { PassageArticle } from "./PassageArticle";
import { fetchCrossref } from "./api";

vi.mock("./api", () => ({ fetchCrossref: vi.fn() }));

const MARKER_HTML =
  'In the beginning<sup><a class="cf" href="Job 38:4-7/">a</a></sup>.';

beforeEach(() => {
  cleanup();
  (fetchCrossref as Mock).mockReset();
});

describe("PassageArticle cross-reference popover", () => {
  it("opens a popover with the referenced verse text on marker click", async () => {
    (fetchCrossref as Mock).mockResolvedValue({
      kind: "ok",
      html: "<p>Where were you when I laid the foundations of the earth?</p>",
      canonical: "Job 38:4-7",
    });
    const { container } = render(<PassageArticle html={MARKER_HTML} showWoc />);

    const marker = container.querySelector("a.cf") as HTMLAnchorElement;
    fireEvent.click(marker);

    const dialog = await screen.findByRole("dialog", {
      name: /cross reference/i,
    });
    expect(dialog).toHaveTextContent(/Where were you/);
    // The bare-reference href is stripped of its trailing slash, not navigated.
    expect(fetchCrossref).toHaveBeenCalledWith("Job 38:4-7");
  });

  it("closes the popover on Escape", async () => {
    (fetchCrossref as Mock).mockResolvedValue({
      kind: "ok",
      html: "<p>verse</p>",
      canonical: "Job 38:4-7",
    });
    const { container } = render(<PassageArticle html={MARKER_HTML} showWoc />);
    fireEvent.click(container.querySelector("a.cf") as HTMLAnchorElement);
    await screen.findByRole("dialog");

    fireEvent.keyDown(document, { key: "Escape" });
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
    );
  });

  it("shows an error message when the lookup fails", async () => {
    (fetchCrossref as Mock).mockResolvedValue({ kind: "error" });
    const { container } = render(<PassageArticle html={MARKER_HTML} showWoc />);
    fireEvent.click(container.querySelector("a.cf") as HTMLAnchorElement);

    const dialog = await screen.findByRole("dialog");
    expect(dialog).toHaveTextContent(/could not load/i);
  });

  it("ignores clicks outside a cross-reference marker", () => {
    render(<PassageArticle html={MARKER_HTML} showWoc />);
    fireEvent.click(screen.getByText(/In the beginning/));
    expect(fetchCrossref).not.toHaveBeenCalled();
  });
});
