import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  createNote,
  deleteNote,
  listNotes,
  type Note,
  updateNote,
} from "./api";

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn());
});

afterEach(() => {
  vi.unstubAllGlobals();
});

const okJSON = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
const statusOnly = (code: number) =>
  new Response(code === 204 || code === 304 ? null : "", { status: code });

const sample: Note = {
  id: 1,
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

describe("listNotes", () => {
  it("returns notes on 200", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      okJSON([sample]),
    );
    const r = await listNotes("John", 3);
    expect(r).toEqual({ kind: "ok", notes: [sample] });
  });
  it("maps 401 to guest", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      statusOnly(401),
    );
    const r = await listNotes("John", 3);
    expect(r.kind).toBe("guest");
  });
  it("maps network failure to error", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockRejectedValueOnce(
      new Error("nope"),
    );
    const r = await listNotes("John", 3);
    expect(r.kind).toBe("error");
  });
});

describe("createNote", () => {
  const input = {
    book: "John",
    chapter: 3,
    start_verse: 16,
    start_offset: 0,
    end_verse: 16,
    end_offset: 10,
    body: "for God so loved",
  };
  it("returns note on 201", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      okJSON(sample, 201),
    );
    const r = await createNote(input);
    expect(r).toEqual({ kind: "ok", note: sample });
  });
  it("maps 400 to invalid", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      statusOnly(400),
    );
    const r = await createNote(input);
    expect(r.kind).toBe("invalid");
  });
  it("maps 401 to guest", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      statusOnly(401),
    );
    const r = await createNote(input);
    expect(r.kind).toBe("guest");
  });
});

describe("updateNote", () => {
  it("returns note on 200", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      okJSON({ ...sample, body: "edited", updated_at: "2026-05-05T00:00:00Z" }),
    );
    const r = await updateNote(1, "edited");
    expect(r.kind).toBe("ok");
    if (r.kind === "ok") {
      expect(r.note.body).toBe("edited");
    }
  });
  it("maps 404 to not_found", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      statusOnly(404),
    );
    const r = await updateNote(1, "x");
    expect(r.kind).toBe("not_found");
  });
  it("maps 400 to invalid", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      statusOnly(400),
    );
    const r = await updateNote(1, "x");
    expect(r.kind).toBe("invalid");
  });
  it("maps 401 to guest", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      statusOnly(401),
    );
    const r = await updateNote(1, "x");
    expect(r.kind).toBe("guest");
  });
});

describe("deleteNote", () => {
  it("returns ok on 204", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      statusOnly(204),
    );
    const r = await deleteNote(1);
    expect(r.kind).toBe("ok");
  });
  it("maps 404 to not_found", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      statusOnly(404),
    );
    const r = await deleteNote(1);
    expect(r.kind).toBe("not_found");
  });
  it("maps 401 to guest", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      statusOnly(401),
    );
    const r = await deleteNote(1);
    expect(r.kind).toBe("guest");
  });
});
