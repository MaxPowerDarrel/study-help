import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createRef } from "react";
import { PassageView } from "./PassageView";
import type { Highlight } from "./api";

const sample: Highlight = {
  id: 42,
  translation: "ESV",
  book: "John",
  chapter: 3,
  start_verse: 16,
  start_offset: 0,
  end_verse: 16,
  end_offset: 11,
  created_at: "2026-05-04T00:00:00Z",
};

const html = [
  `<a class="va" alt="esv_06" rel="v43003016"></a>`,
  `For God so loved the world.`,
].join("");

beforeEach(() => {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(
      new Response(JSON.stringify([sample]), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    ),
  );
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("PassageView click-on-mark", () => {
  it("opens the Remove highlight toolbar when an existing mark is clicked", async () => {
    const articleRef = createRef<HTMLElement | null>();
    render(
      <PassageView
        html={html}
        book="John"
        chapter={3}
        translation="ESV"
        isSignedIn={true}
        showWordsOfChrist={true}
        onGuestSignin={() => {}}
        articleRef={articleRef}
      />,
    );

    // Wait for highlights to load and applyHighlights to wrap the mark.
    const mark = await waitFor(() => {
      const m = articleRef.current?.querySelector("mark.highlight");
      if (!m) throw new Error("mark not yet applied");
      return m as HTMLElement;
    });

    expect(mark.getAttribute("data-highlight-id")).toBe("42");

    // Click the mark — should set mode to "existing" and show the toolbar.
    await userEvent.click(mark);

    // Toolbar shows "Remove highlight" AND the mark stays in the DOM.
    // (Regression: a fresh `dangerouslySetInnerHTML={{__html}}` object on
    // every render used to wipe the mark React didn't know about.)
    expect(
      screen.getByRole("button", { name: /remove highlight/i }),
    ).toBeTruthy();
    expect(articleRef.current?.querySelector("mark.highlight")).toBeTruthy();
  });
});
