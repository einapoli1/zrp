import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import userEvent from "@testing-library/user-event";
import { mockInventory, mockParts } from "../test/mocks";

const mockGetInventory = vi.fn();
const mockGetParts = vi.fn();
const mockCreateInventoryTransaction = vi.fn();
const mockBulkDeleteInventory = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getInventory: (...args: any[]) => mockGetInventory(...args),
    getParts: (...args: any[]) => mockGetParts(...args),
    createInventoryTransaction: (...args: any[]) => mockCreateInventoryTransaction(...args),
    bulkDeleteInventory: (...args: any[]) => mockBulkDeleteInventory(...args),
  },
}));

import Inventory from "./Inventory";

beforeEach(() => {
  vi.clearAllMocks();
  mockGetInventory.mockResolvedValue(mockInventory);
  mockGetParts.mockResolvedValue(mockParts);
  mockCreateInventoryTransaction.mockResolvedValue(undefined);
  mockBulkDeleteInventory.mockResolvedValue(undefined);
});

const waitForLoad = () => waitFor(() => expect(screen.getByText("IPN-001")).toBeInTheDocument());

describe("Inventory", () => {
  it("renders loading state", () => {
    render(<Inventory />);
    expect(screen.getByText("Loading inventory...")).toBeInTheDocument();
  });

  it("renders page title and subtitle", async () => {
    render(<Inventory />);
    await waitForLoad();
    expect(screen.getByText("Inventory")).toBeInTheDocument();
    expect(screen.getByText("Manage your inventory levels and stock tracking.")).toBeInTheDocument();
  });

  it("renders inventory table after loading", async () => {
    render(<Inventory />);
    await waitForLoad();
    expect(screen.getByText("IPN-002")).toBeInTheDocument();
    expect(screen.getByText("IPN-003")).toBeInTheDocument();
  });

  it("shows summary cards", async () => {
    render(<Inventory />);
    await waitForLoad();
    expect(screen.getByText("Total Items")).toBeInTheDocument();
    expect(screen.getByText("Low Stock Items")).toBeInTheDocument();
    expect(screen.getByText("Selected")).toBeInTheDocument();
  });

  it("has quick receive button", async () => {
    render(<Inventory />);
    await waitForLoad();
    expect(screen.getByText("Quick Receive")).toBeInTheDocument();
  });

  it("has low stock filter button", async () => {
    render(<Inventory />);
    await waitForLoad();
    expect(screen.getByText("Low Stock")).toBeInTheDocument();
  });

  it("opens quick receive dialog", async () => {
    render(<Inventory />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Quick Receive"));
    await waitFor(() => {
      expect(screen.getByText("Quick Receive Inventory")).toBeInTheDocument();
    });
  });

  it("quick receive dialog has form fields", async () => {
    render(<Inventory />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Quick Receive"));
    await waitFor(() => {
      expect(screen.getByLabelText("IPN")).toBeInTheDocument();
      expect(screen.getByLabelText("Quantity")).toBeInTheDocument();
      expect(screen.getByLabelText("Reference")).toBeInTheDocument();
      expect(screen.getByLabelText("Notes")).toBeInTheDocument();
    });
  });

  it("shows table headers", async () => {
    render(<Inventory />);
    await waitForLoad();
    expect(screen.getByText("On Hand")).toBeInTheDocument();
    expect(screen.getByText("Reserved")).toBeInTheDocument();
    expect(screen.getByText("Available")).toBeInTheDocument();
    expect(screen.getByText("Location")).toBeInTheDocument();
    expect(screen.getByText("Reorder Point")).toBeInTheDocument();
  });

  it("shows stock levels in table", async () => {
    render(<Inventory />);
    await waitForLoad();
    expect(screen.getByText("500")).toBeInTheDocument();
    expect(screen.getByText("450")).toBeInTheDocument();
  });

  it("shows locations in table", async () => {
    render(<Inventory />);
    await waitForLoad();
    expect(screen.getByText("Bin A1")).toBeInTheDocument();
    expect(screen.getByText("Bin B2")).toBeInTheDocument();
    expect(screen.getByText("Shelf C")).toBeInTheDocument();
  });

  it("shows empty state", async () => {
    mockGetInventory.mockResolvedValue([]);
    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("No inventory items found")).toBeInTheDocument();
    });
  });

  it("toggles low stock filter", async () => {
    render(<Inventory />);
    await waitForLoad();
    mockGetInventory.mockClear();
    fireEvent.click(screen.getByText("Low Stock"));
    await waitFor(() => {
      expect(mockGetInventory).toHaveBeenCalledWith(true);
    });
  });

  it("shows low stock empty state when filtered", async () => {
    render(<Inventory />);
    await waitForLoad();
    mockGetInventory.mockResolvedValue([]);
    fireEvent.click(screen.getByText("Low Stock"));
    await waitFor(() => {
      expect(screen.getByText("No low stock items found")).toBeInTheDocument();
    });
  });

  it("shows bulk actions message when nothing selected", async () => {
    render(<Inventory />);
    await waitForLoad();
    expect(screen.getByText("Select items for bulk actions")).toBeInTheDocument();
  });

  it("renders inventory item links", async () => {
    render(<Inventory />);
    await waitForLoad();
    const link = screen.getByText("IPN-001").closest("a");
    expect(link).toHaveAttribute("href", "/inventory/IPN-001");
  });

  it("cancel closes quick receive dialog", async () => {
    render(<Inventory />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Quick Receive"));
    await waitFor(() => expect(screen.getByText("Quick Receive Inventory")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Cancel"));
    await waitFor(() => {
      expect(screen.queryByText("Quick Receive Inventory")).not.toBeInTheDocument();
    });
  });

  it("shows Inventory Items card title", async () => {
    render(<Inventory />);
    await waitForLoad();
    expect(screen.getByText("Inventory Items")).toBeInTheDocument();
  });

  it("checkbox select and deselect individual items", async () => {
    render(<Inventory />);
    await waitForLoad();

    // Verify nothing selected initially
    expect(screen.getByText("Select items for bulk actions")).toBeInTheDocument();

    // Find checkboxes - first two are select-all (summary card + table header), rest are per-row
    const checkboxes = screen.getAllByRole("checkbox");
    const rowCheckbox = checkboxes[2]; // first row checkbox

    fireEvent.click(rowCheckbox);
    // After selecting one item, "Delete Selected" should appear
    await waitFor(() => {
      expect(screen.getByText("Delete Selected")).toBeInTheDocument();
    });

    // Deselect
    fireEvent.click(rowCheckbox);
    await waitFor(() => {
      expect(screen.getByText("Select items for bulk actions")).toBeInTheDocument();
    });
  });

  it("select-all selects all items and shows Delete Selected", async () => {
    render(<Inventory />);
    await waitForLoad();

    // Click header select-all checkbox (second checkbox, the one in table header)
    const checkboxes = screen.getAllByRole("checkbox");
    const selectAllCheckbox = checkboxes[1]; // table header checkbox
    fireEvent.click(selectAllCheckbox);

    await waitFor(() => {
      expect(screen.getByText("Delete Selected")).toBeInTheDocument();
    });
  });

  it("bulk delete with confirm dialog calls API", async () => {
    // Mock window.confirm
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);

    render(<Inventory />);
    await waitForLoad();

    // Select all
    const checkboxes = screen.getAllByRole("checkbox");
    fireEvent.click(checkboxes[1]); // table header select-all

    await waitFor(() => expect(screen.getByText("Delete Selected")).toBeInTheDocument());

    fireEvent.click(screen.getByText("Delete Selected"));

    await waitFor(() => {
      expect(confirmSpy).toHaveBeenCalledWith("Delete 3 inventory items?");
      expect(mockBulkDeleteInventory).toHaveBeenCalledWith(["IPN-001", "IPN-002", "IPN-003"]);
    });

    confirmSpy.mockRestore();
  });

  it("bulk delete cancelled when confirm is dismissed", async () => {
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);

    render(<Inventory />);
    await waitForLoad();

    const checkboxes = screen.getAllByRole("checkbox");
    fireEvent.click(checkboxes[1]);
    await waitFor(() => expect(screen.getByText("Delete Selected")).toBeInTheDocument());

    fireEvent.click(screen.getByText("Delete Selected"));

    expect(confirmSpy).toHaveBeenCalled();
    expect(mockBulkDeleteInventory).not.toHaveBeenCalled();

    confirmSpy.mockRestore();
  });

  it("quick receive autocomplete dropdown filters by typed IPN", async () => {
    render(<Inventory />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Quick Receive"));
    await waitFor(() => expect(screen.getByLabelText("IPN")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("IPN"), { target: { value: "IPN-00" } });
    const dialog = screen.getByRole("dialog");
    await waitFor(() => {
      const suggestions = dialog.querySelectorAll(".font-medium");
      // All 3 mock parts match "IPN-00"
      expect(suggestions.length).toBeGreaterThanOrEqual(3);
    });
    // Now type a more specific filter
    fireEvent.change(screen.getByLabelText("IPN"), { target: { value: "IPN-003" } });
    await waitFor(() => {
      const suggestions = dialog.querySelectorAll(".hover\\:bg-muted");
      expect(suggestions.length).toBe(1);
    });
  });

  it("selecting from autocomplete populates IPN field", async () => {
    render(<Inventory />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Quick Receive"));
    await waitFor(() => expect(screen.getByLabelText("IPN")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("IPN"), { target: { value: "IPN" } });
    const dialog = screen.getByRole("dialog");
    await waitFor(() => {
      expect(dialog.querySelectorAll(".hover\\:bg-muted").length).toBeGreaterThan(0);
    });
    // Click first suggestion
    const firstSuggestion = dialog.querySelector(".hover\\:bg-muted") as HTMLElement;
    fireEvent.click(firstSuggestion);
    expect(screen.getByLabelText("IPN")).toHaveValue("IPN-001");
  });

  it("displays available qty as Math.max(0, on_hand - reserved)", async () => {
    render(<Inventory />);
    await waitForLoad();
    // IPN-001: 500 - 50 = 450, IPN-002: 20 - 5 = 15, IPN-003: 10 - 0 = 10
    // Check available column (6th td, index 5) per row
    const row1 = screen.getByText("IPN-001").closest("tr")!;
    expect(row1.querySelectorAll("td")[5].textContent).toBe("450");
    const row2 = screen.getByText("IPN-002").closest("tr")!;
    expect(row2.querySelectorAll("td")[5].textContent).toBe("15");
    const row3 = screen.getByText("IPN-003").closest("tr")!;
    expect(row3.querySelectorAll("td")[5].textContent).toBe("10");
  });

  it("displays available qty as 0 when reserved exceeds on_hand", async () => {
    mockGetInventory.mockResolvedValue([
      { ipn: "IPN-NEG", qty_on_hand: 5, qty_reserved: 10, location: "X", reorder_point: 0, reorder_qty: 0, description: "Negative test", updated_at: "2024-01-01" },
    ]);
    render(<Inventory />);
    await waitFor(() => expect(screen.getByText("IPN-NEG")).toBeInTheDocument());
    // Available should be 0, not -5
    const row = screen.getByText("IPN-NEG").closest("tr")!;
    const cells = row.querySelectorAll("td");
    // Available is the 6th cell (index 5)
    expect(cells[5].textContent).toBe("0");
  });

  it("dropdown menu shows View Details and Quick Receive options", async () => {
    const user = userEvent.setup();
    render(<Inventory />);
    await waitForLoad();
    const firstRow = screen.getByText("IPN-001").closest("tr")!;
    const allBtns = firstRow.querySelectorAll("button");
    await user.click(allBtns[allBtns.length - 1]);
    await waitFor(() => {
      expect(screen.getByText("View Details")).toBeInTheDocument();
    });
    const menuItems = screen.getAllByRole("menuitem");
    expect(menuItems.find(m => m.textContent === "Quick Receive")).toBeDefined();
  });

  it("Quick Receive from dropdown pre-fills IPN", async () => {
    const user = userEvent.setup();
    render(<Inventory />);
    await waitForLoad();
    const firstRow = screen.getByText("IPN-001").closest("tr")!;
    const allBtns = firstRow.querySelectorAll("button");
    await user.click(allBtns[allBtns.length - 1]);
    await waitFor(() => expect(screen.getByText("View Details")).toBeInTheDocument());
    const menuItems = screen.getAllByRole("menuitem");
    await user.click(menuItems.find(m => m.textContent === "Quick Receive")!);
    await waitFor(() => expect(screen.getByRole("dialog")).toBeInTheDocument());
    expect(screen.getByLabelText("IPN")).toHaveValue("IPN-001");
  });

  it("highlights low stock items with bg-red-50 class", async () => {
    // IPN-002: qty_on_hand=20, reorder_point=50 → low stock
    render(<Inventory />);
    await waitForLoad();
    const ipn002Row = screen.getByText("IPN-002").closest("tr");
    expect(ipn002Row).toHaveClass("bg-red-50");
  });

  it("does not highlight items above reorder point", async () => {
    // IPN-001: qty_on_hand=500, reorder_point=100 → not low stock
    render(<Inventory />);
    await waitForLoad();
    const ipn001Row = screen.getByText("IPN-001").closest("tr");
    expect(ipn001Row).not.toHaveClass("bg-red-50");
  });

  it("quick receive autocomplete shows matching parts", async () => {
    render(<Inventory />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Quick Receive"));
    await waitFor(() => expect(screen.getByLabelText("IPN")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("IPN"), { target: { value: "IPN" } });
    // Should show autocomplete suggestions inside the dialog
    const dialog = screen.getByRole("dialog");
    await waitFor(() => {
      // Autocomplete creates divs with class "font-medium" containing IPN text
      const suggestions = dialog.querySelectorAll(".hover\\:bg-muted");
      expect(suggestions.length).toBeGreaterThan(0);
    });
  });

  it("quick receive button disabled when IPN and qty empty", async () => {
    render(<Inventory />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Quick Receive"));
    await waitFor(() => expect(screen.getByRole("dialog")).toBeInTheDocument());
    const dialog = screen.getByRole("dialog");
    const receiveBtn = Array.from(dialog.querySelectorAll("button")).find(b => b.textContent === "Receive");
    expect(receiveBtn).toBeDisabled();
  });

  it("submits quick receive form", async () => {
    render(<Inventory />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Quick Receive"));
    await waitFor(() => expect(screen.getByLabelText("IPN")).toBeInTheDocument());

    fireEvent.change(screen.getByLabelText("IPN"), { target: { value: "IPN-001" } });
    fireEvent.change(screen.getByLabelText("Quantity"), { target: { value: "50" } });
    fireEvent.change(screen.getByLabelText("Reference"), { target: { value: "PO-100" } });

    // Find Receive button inside dialog
    const dialog = screen.getByRole("dialog");
    const receiveBtn = Array.from(dialog.querySelectorAll("button")).find(b => b.textContent === "Receive");
    fireEvent.click(receiveBtn!);

    await waitFor(() => {
      expect(mockCreateInventoryTransaction).toHaveBeenCalledWith(
        expect.objectContaining({
          ipn: "IPN-001",
          type: "receive",
          qty: 50,
          reference: "PO-100",
        })
      );
    });
  });
});
