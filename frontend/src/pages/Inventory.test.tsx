import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
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
