import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";

const mockGetUndoList = vi.fn();
const mockPerformUndo = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getUndoList: (...args: unknown[]) => mockGetUndoList(...args),
    performUndo: (...args: unknown[]) => mockPerformUndo(...args),
  },
}));

vi.mock("sonner", () => ({
  toast: Object.assign(vi.fn(), {
    success: vi.fn(),
    error: vi.fn(),
  }),
}));

import UndoHistory from "./UndoHistory";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("UndoHistory", () => {
  it("renders empty state when no entries", async () => {
    mockGetUndoList.mockResolvedValue([]);
    render(<UndoHistory />);
    await waitFor(() => {
      expect(screen.getByText("No undoable actions available")).toBeInTheDocument();
    });
  });

  it("renders undo entries", async () => {
    mockGetUndoList.mockResolvedValue([
      {
        id: 1,
        user_id: "admin",
        action: "delete",
        entity_type: "vendor",
        entity_id: "V-001",
        previous_data: "{}",
        created_at: "2026-02-18 12:00:00",
        expires_at: "2026-02-19 12:00:00",
      },
    ]);
    render(<UndoHistory />);
    await waitFor(() => {
      expect(screen.getByText("V-001")).toBeInTheDocument();
    });
    expect(screen.getByText(/delete/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /undo/i })).toBeInTheDocument();
  });

  it("calls performUndo when Undo button clicked", async () => {
    mockGetUndoList.mockResolvedValue([
      {
        id: 5,
        user_id: "admin",
        action: "delete",
        entity_type: "eco",
        entity_id: "ECO-001",
        previous_data: "{}",
        created_at: "2026-02-18 12:00:00",
        expires_at: "2026-02-19 12:00:00",
      },
    ]);
    mockPerformUndo.mockResolvedValue({ status: "restored" });

    render(<UndoHistory />);
    await waitFor(() => {
      expect(screen.getByText("ECO-001")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: /undo/i }));
    await waitFor(() => {
      expect(mockPerformUndo).toHaveBeenCalledWith(5);
    });
  });

  it("shows page heading", async () => {
    mockGetUndoList.mockResolvedValue([]);
    render(<UndoHistory />);
    await waitFor(() => {
      expect(screen.getByText("Undo History")).toBeInTheDocument();
    });
  });
});
