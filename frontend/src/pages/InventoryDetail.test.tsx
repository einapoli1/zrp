import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockInventory } from "../test/mocks";
import type { InventoryTransaction } from "../lib/api";

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ ipn: "IPN-001" }),
  };
});

const mockItem = mockInventory[0]; // IPN-001, qty_on_hand=500, qty_reserved=50, reorder_point=100

const mockTransactions: InventoryTransaction[] = [
  { id: 1, ipn: "IPN-001", type: "receive", qty: 500, reference: "PO-001", notes: "Initial stock", created_at: "2024-01-15T10:00:00Z" },
  { id: 2, ipn: "IPN-001", type: "issue", qty: 50, reference: "WO-001", notes: "Production run", created_at: "2024-01-18T14:30:00Z" },
  { id: 3, ipn: "IPN-001", type: "adjust", qty: 500, reference: "", notes: "Cycle count", created_at: "2024-01-20T09:00:00Z" },
];

const mockGetInventoryItem = vi.fn();
const mockGetInventoryHistory = vi.fn();
const mockCreateInventoryTransaction = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getInventoryItem: (...args: any[]) => mockGetInventoryItem(...args),
    getInventoryHistory: (...args: any[]) => mockGetInventoryHistory(...args),
    createInventoryTransaction: (...args: any[]) => mockCreateInventoryTransaction(...args),
  },
}));

import InventoryDetail from "./InventoryDetail";

beforeEach(() => {
  vi.clearAllMocks();
  mockGetInventoryItem.mockResolvedValue(mockItem);
  mockGetInventoryHistory.mockResolvedValue(mockTransactions);
  mockCreateInventoryTransaction.mockResolvedValue(undefined);
});

const waitForLoad = () => waitFor(() => expect(screen.getByText("Item Details")).toBeInTheDocument());

describe("InventoryDetail", () => {
  it("renders loading state initially", () => {
    mockGetInventoryItem.mockReturnValue(new Promise(() => {}));
    render(<InventoryDetail />);
    expect(screen.getByText("Loading inventory item...")).toBeInTheDocument();
  });

  it("renders item IPN as heading", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("IPN-001");
  });

  it("renders item description", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    // Description appears in header and details section
    expect(screen.getAllByText("10k Resistor").length).toBeGreaterThan(0);
  });

  it("renders back to inventory link", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    expect(screen.getByText("Back to Inventory")).toBeInTheDocument();
    const link = screen.getByText("Back to Inventory").closest("a");
    expect(link).toHaveAttribute("href", "/inventory");
  });

  it("renders stock level cards", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    expect(screen.getByText("On Hand")).toBeInTheDocument();
    expect(screen.getByText("Reserved")).toBeInTheDocument();
    expect(screen.getByText("Available")).toBeInTheDocument();
    expect(screen.getByText("Reorder Point")).toBeInTheDocument();
  });

  it("shows correct available qty (on_hand - reserved)", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    expect(screen.getByText("450")).toBeInTheDocument(); // 500 - 50
  });

  it("renders Item Details card", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    expect(screen.getByText("Item Details")).toBeInTheDocument();
    expect(screen.getByText("Internal Part Number")).toBeInTheDocument();
  });

  it("shows location in details", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    expect(screen.getByText("Bin A1")).toBeInTheDocument();
  });

  it("shows reorder quantity", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    expect(screen.getByText("Reorder Quantity")).toBeInTheDocument();
  });

  it("renders Transaction History card", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    expect(screen.getByText("Transaction History")).toBeInTheDocument();
  });

  it("shows transaction table headers", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    expect(screen.getByText("Date")).toBeInTheDocument();
    expect(screen.getByText("Type")).toBeInTheDocument();
    expect(screen.getByText("Quantity")).toBeInTheDocument();
    expect(screen.getByText("Reference")).toBeInTheDocument();
    expect(screen.getByText("Notes")).toBeInTheDocument();
  });

  it("shows transaction types as badges", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    expect(screen.getByText("RECEIVE")).toBeInTheDocument();
    expect(screen.getByText("ISSUE")).toBeInTheDocument();
    expect(screen.getByText("ADJUST")).toBeInTheDocument();
  });

  it("shows transaction references", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    expect(screen.getByText("PO-001")).toBeInTheDocument();
    expect(screen.getByText("WO-001")).toBeInTheDocument();
  });

  it("shows transaction notes", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    expect(screen.getByText("Initial stock")).toBeInTheDocument();
    expect(screen.getByText("Production run")).toBeInTheDocument();
    expect(screen.getByText("Cycle count")).toBeInTheDocument();
  });

  it("shows issue qty with negative sign", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    expect(screen.getByText("-50")).toBeInTheDocument();
  });

  it("shows empty transaction history message", async () => {
    mockGetInventoryHistory.mockResolvedValue([]);
    render(<InventoryDetail />);
    await waitForLoad();
    expect(screen.getByText("No transaction history found for this item")).toBeInTheDocument();
  });

  it("has New Transaction button", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    expect(screen.getByText("New Transaction")).toBeInTheDocument();
  });

  it("opens transaction dialog", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    fireEvent.click(screen.getByText("New Transaction"));
    await waitFor(() => {
      expect(screen.getByText("Create Inventory Transaction")).toBeInTheDocument();
    });
  });

  it("transaction dialog has form fields", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    fireEvent.click(screen.getByText("New Transaction"));
    await waitFor(() => {
      expect(screen.getByLabelText("Quantity")).toBeInTheDocument();
      expect(screen.getByLabelText("Reference")).toBeInTheDocument();
      expect(screen.getByLabelText("Notes")).toBeInTheDocument();
      expect(screen.getByText("Transaction Type")).toBeInTheDocument();
    });
  });

  it("create transaction button disabled when qty empty", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    fireEvent.click(screen.getByText("New Transaction"));
    await waitFor(() => {
      expect(screen.getByText("Create Transaction")).toBeDisabled();
    });
  });

  it("submits transaction form", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    fireEvent.click(screen.getByText("New Transaction"));
    await waitFor(() => expect(screen.getByLabelText("Quantity")).toBeInTheDocument());

    fireEvent.change(screen.getByLabelText("Quantity"), { target: { value: "25" } });
    fireEvent.change(screen.getByLabelText("Reference"), { target: { value: "PO-999" } });
    fireEvent.click(screen.getByText("Create Transaction"));

    await waitFor(() => {
      expect(mockCreateInventoryTransaction).toHaveBeenCalledWith(
        expect.objectContaining({
          ipn: "IPN-001",
          type: "receive",
          qty: 25,
          reference: "PO-999",
        })
      );
    });
  });

  it("cancel closes transaction dialog", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    fireEvent.click(screen.getByText("New Transaction"));
    await waitFor(() => expect(screen.getByText("Create Inventory Transaction")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Cancel"));
    await waitFor(() => {
      expect(screen.queryByText("Create Inventory Transaction")).not.toBeInTheDocument();
    });
  });

  it("shows not found when item is null", async () => {
    mockGetInventoryItem.mockResolvedValue(null);
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("Inventory Item Not Found")).toBeInTheDocument();
    });
  });

  it("does not show LOW badge when stock is above reorder point", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    expect(screen.queryByText("LOW")).not.toBeInTheDocument();
  });

  it("can select issue transaction type and submit", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    fireEvent.click(screen.getByText("New Transaction"));
    await waitFor(() => expect(screen.getByText("Create Inventory Transaction")).toBeInTheDocument());
    // Open transaction type select
    const typeLabel = screen.getByText("Transaction Type");
    const selectTrigger = typeLabel.parentElement?.querySelector("[role='combobox']") as HTMLElement;
    fireEvent.click(selectTrigger);
    await waitFor(() => expect(screen.getByText("Issue")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Issue"));
    fireEvent.change(screen.getByLabelText("Quantity"), { target: { value: "10" } });
    fireEvent.click(screen.getByText("Create Transaction"));
    await waitFor(() => {
      expect(mockCreateInventoryTransaction).toHaveBeenCalledWith(
        expect.objectContaining({ type: "issue", qty: 10 })
      );
    });
  });

  it("can select adjust transaction type", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    fireEvent.click(screen.getByText("New Transaction"));
    await waitFor(() => expect(screen.getByText("Create Inventory Transaction")).toBeInTheDocument());
    const selectTrigger = screen.getByText("Transaction Type").parentElement?.querySelector("[role='combobox']") as HTMLElement;
    fireEvent.click(selectTrigger);
    await waitFor(() => expect(screen.getByText("Adjust")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Adjust"));
    // Should show adjustment help text
    fireEvent.change(screen.getByLabelText("Quantity"), { target: { value: "100" } });
    fireEvent.click(screen.getByText("Create Transaction"));
    await waitFor(() => {
      expect(mockCreateInventoryTransaction).toHaveBeenCalledWith(
        expect.objectContaining({ type: "adjust", qty: 100 })
      );
    });
  });

  it("can select return transaction type", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    fireEvent.click(screen.getByText("New Transaction"));
    await waitFor(() => expect(screen.getByText("Create Inventory Transaction")).toBeInTheDocument());
    const selectTrigger = screen.getByText("Transaction Type").parentElement?.querySelector("[role='combobox']") as HTMLElement;
    fireEvent.click(selectTrigger);
    await waitFor(() => expect(screen.getByText("Return")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Return"));
    fireEvent.change(screen.getByLabelText("Quantity"), { target: { value: "5" } });
    fireEvent.click(screen.getByText("Create Transaction"));
    await waitFor(() => {
      expect(mockCreateInventoryTransaction).toHaveBeenCalledWith(
        expect.objectContaining({ type: "return", qty: 5 })
      );
    });
  });

  it("shows adjust type hint text when adjust is selected", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    fireEvent.click(screen.getByText("New Transaction"));
    await waitFor(() => expect(screen.getByText("Create Inventory Transaction")).toBeInTheDocument());
    const selectTrigger = screen.getByText("Transaction Type").parentElement?.querySelector("[role='combobox']") as HTMLElement;
    fireEvent.click(selectTrigger);
    await waitFor(() => expect(screen.getByText("Adjust")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Adjust"));
    expect(screen.getByText("For adjustments, enter the new total quantity")).toBeInTheDocument();
  });

  it("handles transaction API error gracefully", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    mockCreateInventoryTransaction.mockRejectedValue(new Error("Server error"));
    render(<InventoryDetail />);
    await waitForLoad();
    fireEvent.click(screen.getByText("New Transaction"));
    await waitFor(() => expect(screen.getByLabelText("Quantity")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("Quantity"), { target: { value: "10" } });
    fireEvent.click(screen.getByText("Create Transaction"));
    await waitFor(() => {
      expect(consoleSpy).toHaveBeenCalledWith("Failed to create transaction:", expect.any(Error));
    });
    // Dialog should still be open (transaction failed, no close)
    expect(screen.getByText("Create Inventory Transaction")).toBeInTheDocument();
    consoleSpy.mockRestore();
  });

  it("refreshes detail and history after successful transaction", async () => {
    render(<InventoryDetail />);
    await waitForLoad();
    // Clear call counts after initial load
    mockGetInventoryItem.mockClear();
    mockGetInventoryHistory.mockClear();
    fireEvent.click(screen.getByText("New Transaction"));
    await waitFor(() => expect(screen.getByLabelText("Quantity")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("Quantity"), { target: { value: "25" } });
    fireEvent.click(screen.getByText("Create Transaction"));
    await waitFor(() => {
      expect(mockGetInventoryItem).toHaveBeenCalledWith("IPN-001");
      expect(mockGetInventoryHistory).toHaveBeenCalledWith("IPN-001");
    });
  });

  it("shows LOW badge when stock is at or below reorder point", async () => {
    mockGetInventoryItem.mockResolvedValue({
      ...mockItem,
      qty_on_hand: 50,
      reorder_point: 100,
    });
    render(<InventoryDetail />);
    await waitForLoad();
    expect(screen.getByText("LOW")).toBeInTheDocument();
  });
});
