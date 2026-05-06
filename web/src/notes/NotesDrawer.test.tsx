import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { NotesDrawer } from "./NotesDrawer";
import type { Note } from "./api";

const note: Note = {
  id: 1,
  translation: "ESV",
  book: "John",
  chapter: 3,
  start_verse: 16,
  start_offset: 0,
  end_verse: 16,
  end_offset: 10,
  body: "for God so loved",
  created_at: "2026-05-04T00:00:00Z",
  updated_at: "2026-05-04T00:00:00Z",
};

const defaultProps = {
  open: true,
  onClose: vi.fn(),
  title: "John 3",
  notes: [] as Note[],
  loading: false,
  pendingTuple: null,
  onCancelPending: vi.fn(),
  onCreate: vi.fn(),
  onUpdate: vi.fn(),
  onRemove: vi.fn(),
  onScrollToAnchor: vi.fn(),
  onError: vi.fn(),
};

describe("NotesDrawer", () => {
  it("renders nothing when closed", () => {
    const { container } = render(
      <NotesDrawer {...defaultProps} open={false} />,
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("renders empty state silently when no notes and not loading", () => {
    render(<NotesDrawer {...defaultProps} />);
    expect(screen.getByRole("dialog", { name: "Notes" })).toBeInTheDocument();
    expect(screen.queryByRole("listitem")).not.toBeInTheDocument();
  });

  it("renders the spinner while loading", () => {
    render(<NotesDrawer {...defaultProps} loading />);
    expect(screen.getByLabelText("Loading")).toBeInTheDocument();
  });

  it("renders entries with passage refs and bodies", () => {
    render(<NotesDrawer {...defaultProps} notes={[note]} />);
    expect(screen.getByText("Verse 16")).toBeInTheDocument();
    expect(screen.getByText("for God so loved")).toBeInTheDocument();
    expect(screen.queryByText("edited")).not.toBeInTheDocument();
  });

  it("renders 'edited' tag when updated_at differs from created_at", () => {
    const edited = { ...note, updated_at: "2026-05-05T00:00:00Z" };
    render(<NotesDrawer {...defaultProps} notes={[edited]} />);
    expect(screen.getByText("edited")).toBeInTheDocument();
  });

  it("calls onScrollToAnchor when entry body is clicked", () => {
    const onScrollToAnchor = vi.fn();
    render(
      <NotesDrawer
        {...defaultProps}
        notes={[note]}
        onScrollToAnchor={onScrollToAnchor}
      />,
    );
    fireEvent.click(screen.getByText("for God so loved"));
    expect(onScrollToAnchor).toHaveBeenCalledWith(note);
  });

  it("enters edit mode and calls onUpdate on save", async () => {
    const onUpdate = vi.fn().mockResolvedValue({ kind: "ok", note });
    render(
      <NotesDrawer {...defaultProps} notes={[note]} onUpdate={onUpdate} />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Edit" }));
    const textarea = screen.getByRole("textbox") as HTMLTextAreaElement;
    fireEvent.change(textarea, { target: { value: "edited body" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    expect(onUpdate).toHaveBeenCalledWith(1, "edited body");
  });

  it("calls onRemove when Delete is clicked", () => {
    const onRemove = vi.fn().mockResolvedValue({ kind: "ok" });
    render(
      <NotesDrawer {...defaultProps} notes={[note]} onRemove={onRemove} />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    expect(onRemove).toHaveBeenCalledWith(1);
  });

  it("renders the composer when pendingTuple is set", () => {
    const tuple = {
      start_verse: 16,
      start_offset: 0,
      end_verse: 17,
      end_offset: 5,
    };
    render(<NotesDrawer {...defaultProps} pendingTuple={tuple} />);
    expect(screen.getByText("Verses 16–17")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Write a note…")).toBeInTheDocument();
  });

  it("composer Save calls onCreate with the typed body", () => {
    const onCreate = vi.fn().mockResolvedValue({ kind: "ok", note });
    const tuple = {
      start_verse: 16,
      start_offset: 0,
      end_verse: 16,
      end_offset: 10,
    };
    render(
      <NotesDrawer
        {...defaultProps}
        pendingTuple={tuple}
        onCreate={onCreate}
      />,
    );
    fireEvent.change(screen.getByPlaceholderText("Write a note…"), {
      target: { value: "fresh note" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(onCreate).toHaveBeenCalledWith("fresh note");
  });

  it("composer Cancel calls onCancelPending", () => {
    const onCancelPending = vi.fn();
    const tuple = {
      start_verse: 16,
      start_offset: 0,
      end_verse: 16,
      end_offset: 10,
    };
    render(
      <NotesDrawer
        {...defaultProps}
        pendingTuple={tuple}
        onCancelPending={onCancelPending}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onCancelPending).toHaveBeenCalled();
  });
});
