import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";

const mockBackups = [
  { filename: "zrp-backup-2025-01-15T10-00-00.db", size: 1048576, created_at: "2025-01-15T10:00:00Z" },
  { filename: "zrp-backup-2025-01-14T10-00-00.db", size: 524288, created_at: "2025-01-14T10:00:00Z" },
];

const mockGetBackups = vi.fn().mockResolvedValue(mockBackups);
const mockCreateBackup = vi.fn().mockResolvedValue(undefined);
const mockDeleteBackup = vi.fn().mockResolvedValue(undefined);
const mockRestoreBackup = vi.fn().mockResolvedValue(undefined);

vi.mock("../lib/api", () => ({
  api: {
    getBackups: (...args: any[]) => mockGetBackups(...args),
    createBackup: (...args: any[]) => mockCreateBackup(...args),
    deleteBackup: (...args: any[]) => mockDeleteBackup(...args),
    restoreBackup: (...args: any[]) => mockRestoreBackup(...args),
  },
}));

import Backups from "./Backups";

beforeEach(() => vi.clearAllMocks());

describe("Backups", () => {
  it("renders backup list after loading", async () => {
    render(<Backups />);
    await waitFor(() => {
      expect(screen.getByText("zrp-backup-2025-01-15T10-00-00.db")).toBeInTheDocument();
    });
    expect(screen.getByText("zrp-backup-2025-01-14T10-00-00.db")).toBeInTheDocument();
  });

  it("shows backup count badge", async () => {
    render(<Backups />);
    await waitFor(() => {
      expect(screen.getByText("2")).toBeInTheDocument();
    });
  });

  it("shows empty state when no backups", async () => {
    mockGetBackups.mockResolvedValueOnce([]);
    render(<Backups />);
    await waitFor(() => {
      expect(screen.getByText(/No backups yet/)).toBeInTheDocument();
    });
  });

  it("has Create Backup button", async () => {
    render(<Backups />);
    await waitFor(() => {
      expect(screen.getByText("Create Backup")).toBeInTheDocument();
    });
  });

  it("calls createBackup and refreshes list", async () => {
    render(<Backups />);
    await waitFor(() => {
      expect(screen.getByText("zrp-backup-2025-01-15T10-00-00.db")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("Create Backup"));
    await waitFor(() => {
      expect(mockCreateBackup).toHaveBeenCalledTimes(1);
    });
    // Should re-fetch backups after create
    expect(mockGetBackups).toHaveBeenCalledTimes(2);
  });

  it("download button opens backup URL", async () => {
    const openSpy = vi.fn();
    window.open = openSpy;

    render(<Backups />);
    await waitFor(() => {
      expect(screen.getByText("zrp-backup-2025-01-15T10-00-00.db")).toBeInTheDocument();
    });

    // Find download buttons (first one in each row)
    const downloadButtons = screen.getAllByRole("button").filter(
      (btn) => btn.querySelector("svg.lucide-download")
    );
    expect(downloadButtons.length).toBeGreaterThan(0);
    fireEvent.click(downloadButtons[0]);
    expect(openSpy).toHaveBeenCalledWith(
      "/api/v1/admin/backups/zrp-backup-2025-01-15T10-00-00.db",
      "_blank"
    );
  });

  it("delete requires confirmation", async () => {
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);

    render(<Backups />);
    await waitFor(() => {
      expect(screen.getByText("zrp-backup-2025-01-15T10-00-00.db")).toBeInTheDocument();
    });

    const deleteButtons = screen.getAllByRole("button").filter(
      (btn) => btn.querySelector("svg.lucide-trash-2")
    );
    fireEvent.click(deleteButtons[0]);
    expect(confirmSpy).toHaveBeenCalled();
    expect(mockDeleteBackup).not.toHaveBeenCalled();

    confirmSpy.mockRestore();
  });

  it("delete calls API when confirmed", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);

    render(<Backups />);
    await waitFor(() => {
      expect(screen.getByText("zrp-backup-2025-01-15T10-00-00.db")).toBeInTheDocument();
    });

    const deleteButtons = screen.getAllByRole("button").filter(
      (btn) => btn.querySelector("svg.lucide-trash-2")
    );
    fireEvent.click(deleteButtons[0]);
    await waitFor(() => {
      expect(mockDeleteBackup).toHaveBeenCalledWith("zrp-backup-2025-01-15T10-00-00.db");
    });
  });

  it("restore requires confirmation", async () => {
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);

    render(<Backups />);
    await waitFor(() => {
      expect(screen.getByText("zrp-backup-2025-01-15T10-00-00.db")).toBeInTheDocument();
    });

    const restoreButtons = screen.getAllByRole("button").filter(
      (btn) => btn.querySelector("svg.lucide-rotate-ccw")
    );
    fireEvent.click(restoreButtons[0]);
    expect(confirmSpy).toHaveBeenCalled();
    expect(mockRestoreBackup).not.toHaveBeenCalled();

    confirmSpy.mockRestore();
  });

  it("restore calls API when confirmed", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    vi.spyOn(window, "alert").mockImplementation(() => {});
    // Mock reload
    const reloadMock = vi.fn();
    Object.defineProperty(window, "location", {
      value: { ...window.location, reload: reloadMock },
      writable: true,
    });

    render(<Backups />);
    await waitFor(() => {
      expect(screen.getByText("zrp-backup-2025-01-15T10-00-00.db")).toBeInTheDocument();
    });

    const restoreButtons = screen.getAllByRole("button").filter(
      (btn) => btn.querySelector("svg.lucide-rotate-ccw")
    );
    fireEvent.click(restoreButtons[0]);
    await waitFor(() => {
      expect(mockRestoreBackup).toHaveBeenCalledWith("zrp-backup-2025-01-15T10-00-00.db");
    });
  });

  it("shows error on API failure", async () => {
    mockGetBackups.mockRejectedValueOnce(new Error("Network error"));
    render(<Backups />);
    await waitFor(() => {
      expect(screen.getByText("Network error")).toBeInTheDocument();
    });
  });
});
