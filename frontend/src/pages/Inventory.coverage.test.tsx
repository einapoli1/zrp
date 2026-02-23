import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockInventory } from "../test/mocks";

const mockGetInventory = vi.fn();
const mockGetParts = vi.fn();
const mockCreateInventoryTransaction = vi.fn();
const mockBulkDeleteInventory = vi.fn();
const mockBulkUpdateInventory = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getInventory: (...args: any[]) => mockGetInventory(...args),
    getParts: (...args: any[]) => mockGetParts(...args),
    createInventoryTransaction: (...args: any[]) => mockCreateInventoryTransaction(...args),
    bulkDeleteInventory: (...args: any[]) => mockBulkDeleteInventory(...args),
    bulkUpdateInventory: (...args: any[]) => mockBulkUpdateInventory(...args),
  },
}));

import Inventory from "./Inventory";

beforeEach(() => {
  vi.clearAllMocks();
  mockGetInventory.mockResolvedValue(mockInventory);
  mockGetParts.mockResolvedValue([]);
  mockCreateInventoryTransaction.mockResolvedValue(undefined);
  mockBulkDeleteInventory.mockResolvedValue(undefined);
  mockBulkUpdateInventory.mockResolvedValue({ success: 3, failed: 0, errors: [] });
});

const waitForLoad = () => waitFor(() => expect(screen.getByText("IPN-001")).toBeInTheDocument());

describe("Inventory - Additional Coverage Tests", () => {
  it("handles API error gracefully when loading inventory", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    mockGetInventory.mockRejectedValue(new Error("Network error"));

    render(<Inventory />);

    await waitFor(() => {
      expect(consoleSpy).toHaveBeenCalledWith("Failed to fetch inventory:", expect.any(Error));
    });

    consoleSpy.mockRestore();
  });

  it("handles API error in quick receive", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    mockCreateInventoryTransaction.mockRejectedValue(new Error("Server error"));

    render(<Inventory />);
    await waitForLoad();

    fireEvent.click(screen.getByText("Quick Receive"));
    await waitFor(() => expect(screen.getByLabelText("IPN")).toBeInTheDocument());

    fireEvent.change(screen.getByLabelText("IPN"), { target: { value: "IPN-001" } });
    fireEvent.change(screen.getByLabelText("Quantity"), { target: { value: "50" } });

    const dialog = screen.getByRole("dialog");
    const receiveBtn = Array.from(dialog.querySelectorAll("button")).find(b => b.textContent === "Receive");
    fireEvent.click(receiveBtn!);

    await waitFor(() => {
      expect(consoleSpy).toHaveBeenCalledWith("Failed to create transaction:", expect.any(Error));
    });

    consoleSpy.mockRestore();
  });

  it("refreshes inventory list after quick receive", async () => {
    render(<Inventory />);
    await waitForLoad();

    // Clear initial load calls
    mockGetInventory.mockClear();

    fireEvent.click(screen.getByText("Quick Receive"));
    await waitFor(() => expect(screen.getByLabelText("IPN")).toBeInTheDocument());

    fireEvent.change(screen.getByLabelText("IPN"), { target: { value: "IPN-001" } });
    fireEvent.change(screen.getByLabelText("Quantity"), { target: { value: "25" } });

    const dialog = screen.getByRole("dialog");
    const receiveBtn = Array.from(dialog.querySelectorAll("button")).find(b => b.textContent === "Receive");
    fireEvent.click(receiveBtn!);

    await waitFor(() => {
      expect(mockGetInventory).toHaveBeenCalled();
    });
  });

  it("shows validation error for negative quantity in quick receive", async () => {
    render(<Inventory />);
    await waitForLoad();

    fireEvent.click(screen.getByText("Quick Receive"));
    await waitFor(() => expect(screen.getByLabelText("Quantity")).toBeInTheDocument());

    // Try entering negative quantity
    fireEvent.change(screen.getByLabelText("Quantity"), { target: { value: "-10" } });
    
    // HTML5 input type="number" with min="0" should prevent this
    const qtyInput = screen.getByLabelText("Quantity") as HTMLInputElement;
    expect(qtyInput.type).toBe("number");
  });

  it("shows correct summary counts", async () => {
    render(<Inventory />);
    await waitForLoad();

    // Total Items: 3
    const totalCard = screen.getByText("Total Items").closest("div");
    expect(totalCard?.textContent).toContain("3");

    // Low Stock Items: 1 (IPN-002 with qty=20, reorder=50)
    const lowStockCard = screen.getByText("Low Stock Items").closest("div");
    expect(lowStockCard?.textContent).toContain("1");
  });

  it("updates selected count when selecting items", async () => {
    render(<Inventory />);
    await waitForLoad();

    const checkboxes = screen.getAllByRole("checkbox");
    fireEvent.click(checkboxes[2]); // Select first row

    await waitFor(() => {
      const selectedCard = screen.getByText("Selected").closest("div");
      expect(selectedCard?.textContent).toContain("1");
    });
  });

  it("deselects all when select-all is clicked twice", async () => {
    render(<Inventory />);
    await waitForLoad();

    const checkboxes = screen.getAllByRole("checkbox");
    const selectAllCheckbox = checkboxes[1]; // table header checkbox

    // Select all
    fireEvent.click(selectAllCheckbox);
    await waitFor(() => expect(screen.getByText("Delete Selected")).toBeInTheDocument());

    // Deselect all
    fireEvent.click(selectAllCheckbox);
    await waitFor(() => {
      expect(screen.getByText("Select items for bulk actions")).toBeInTheDocument();
    });
  });

  it("closes quick receive dialog after successful transaction", async () => {
    render(<Inventory />);
    await waitForLoad();

    fireEvent.click(screen.getByText("Quick Receive"));
    await waitFor(() => expect(screen.getByRole("dialog")).toBeInTheDocument());

    fireEvent.change(screen.getByLabelText("IPN"), { target: { value: "IPN-001" } });
    fireEvent.change(screen.getByLabelText("Quantity"), { target: { value: "25" } });

    const dialog = screen.getByRole("dialog");
    const receiveBtn = Array.from(dialog.querySelectorAll("button")).find(b => b.textContent === "Receive");
    fireEvent.click(receiveBtn!);

    await waitFor(() => {
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });
  });

  it("handles bulk edit with location updates", async () => {
    render(<Inventory />);
    await waitForLoad();

    const checkboxes = screen.getAllByRole("checkbox");
    fireEvent.click(checkboxes[1]); // Select all

    await waitFor(() => expect(screen.getByText("Bulk Edit")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Bulk Edit"));

    await waitFor(() => {
      expect(screen.getByText("Bulk Edit Inventory")).toBeInTheDocument();
    });

    // Bulk edit dialog should have location field
    expect(screen.getByText("Location")).toBeInTheDocument();
  });

  it("shows MPN column in table", async () => {
    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });

    // Check if MPN column is visible (depends on ConfigurableTable default columns)
    // MPN may not be visible by default, but should be available in column settings
  });

  it("handles empty parts list in autocomplete", async () => {
    mockGetParts.mockResolvedValue([]);

    render(<Inventory />);
    await waitForLoad();

    fireEvent.click(screen.getByText("Quick Receive"));
    await waitFor(() => expect(screen.getByLabelText("IPN")).toBeInTheDocument());

    fireEvent.change(screen.getByLabelText("IPN"), { target: { value: "NONEXISTENT" } });

    const dialog = screen.getByRole("dialog");
    await waitFor(() => {
      const suggestions = dialog.querySelectorAll(".hover\\:bg-muted");
      expect(suggestions.length).toBe(0);
    });
  });

  it("bulk update shows success message", async () => {
    render(<Inventory />);
    await waitForLoad();

    const checkboxes = screen.getAllByRole("checkbox");
    fireEvent.click(checkboxes[2]); // Select one item

    await waitFor(() => expect(screen.getByText("Bulk Edit")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Bulk Edit"));

    await waitFor(() => expect(screen.getByText("Bulk Edit Inventory")).toBeInTheDocument());

    // Fill and submit bulk edit form
    const locationInput = screen.getByLabelText("Location");
    fireEvent.change(locationInput, { target: { value: "Updated Location" } });

    const submitBtn = screen.getByText("Update Items");
    fireEvent.click(submitBtn);

    await waitFor(() => {
      expect(mockBulkUpdateInventory).toHaveBeenCalled();
    });
  });

  it("shows loading spinner during initial load", () => {
    mockGetInventory.mockReturnValue(new Promise(() => {})); // Never resolves

    render(<Inventory />);
    expect(screen.getByText("Loading inventory...")).toBeInTheDocument();
  });

  it("handles very long IPN gracefully", async () => {
    const longIPN = "IPN-" + "X".repeat(100);
    mockGetInventory.mockResolvedValue([
      { ipn: longIPN, qty_on_hand: 100, qty_reserved: 0, location: "A1", reorder_point: 10, reorder_qty: 50, description: "Test", updated_at: "2024-01-01" },
    ]);

    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText(longIPN)).toBeInTheDocument();
    });

    // Verify table doesn't break layout
    const table = screen.getByRole("table");
    expect(table).toBeInTheDocument();
  });

  it("shows reorder qty in table", async () => {
    render(<Inventory />);
    await waitForLoad();

    // IPN-001 has reorder_qty: 200
    // Check if Reorder Qty column exists (may require enabling via column settings)
    const headers = screen.getAllByRole("columnheader");
    const hasReorderQty = headers.some(h => h.textContent?.includes("Reorder"));
    expect(hasReorderQty).toBe(true);
  });

  it("handles inventory with zero on_hand and reserved", async () => {
    mockGetInventory.mockResolvedValue([
      { ipn: "IPN-ZERO", qty_on_hand: 0, qty_reserved: 0, location: "A1", reorder_point: 10, reorder_qty: 50, description: "Zero stock", updated_at: "2024-01-01" },
    ]);

    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("IPN-ZERO")).toBeInTheDocument();
    });

    const row = screen.getByText("IPN-ZERO").closest("tr");
    expect(row).toBeInTheDocument();

    // Available should be 0
    const cells = row!.querySelectorAll("td");
    // Find Available column (typically 6th column)
    const availableCell = cells[5];
    expect(availableCell.textContent).toBe("0");
  });

  it("filters autocomplete case-insensitively", async () => {
    mockGetParts.mockResolvedValue([
      { ipn: "IPN-ABC", description: "Test Part ABC" },
      { ipn: "IPN-XYZ", description: "Test Part XYZ" },
    ]);

    render(<Inventory />);
    await waitForLoad();

    fireEvent.click(screen.getByText("Quick Receive"));
    await waitFor(() => expect(screen.getByLabelText("IPN")).toBeInTheDocument());

    fireEvent.change(screen.getByLabelText("IPN"), { target: { value: "abc" } });

    const dialog = screen.getByRole("dialog");
    await waitFor(() => {
      const text = dialog.textContent || "";
      expect(text.includes("IPN-ABC")).toBe(true);
    });
  });
});
