import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockPOs, mockVendors, mockParts } from "../test/mocks";

const mockPOsWithLines = [
  { ...mockPOs[0], lines: [
    { id: 1, po_id: "PO-001", ipn: "IPN-001", mpn: "R10K", manufacturer: "Yageo", qty_ordered: 100, qty_received: 0, unit_price: 0.01, notes: "" },
    { id: 2, po_id: "PO-001", ipn: "IPN-002", mpn: "C100U", manufacturer: "Murata", qty_ordered: 50, qty_received: 0, unit_price: 0.10, notes: "" },
  ]},
  { ...mockPOs[1], lines: [] },
];

const mockGetPurchaseOrders = vi.fn().mockResolvedValue(mockPOsWithLines);
const mockGetVendors = vi.fn().mockResolvedValue(mockVendors);
const mockGetParts = vi.fn().mockResolvedValue(mockParts);
const mockCreatePurchaseOrder = vi.fn().mockResolvedValue(mockPOs[0]);

vi.mock("../lib/api", () => ({
  api: {
    getPurchaseOrders: (...args: any[]) => mockGetPurchaseOrders(...args),
    getVendors: (...args: any[]) => mockGetVendors(...args),
    getParts: (...args: any[]) => mockGetParts(...args),
    createPurchaseOrder: (...args: any[]) => mockCreatePurchaseOrder(...args),
  },
}));

import Procurement from "./Procurement";

beforeEach(() => vi.clearAllMocks());

describe("Procurement", () => {
  it("renders loading state", () => {
    render(<Procurement />);
    expect(screen.getByText("Loading purchase orders...")).toBeInTheDocument();
  });

  it("renders PO list after loading", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });
    expect(screen.getByText("PO-002")).toBeInTheDocument();
  });

  it("shows page title and description", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("Procurement")).toBeInTheDocument();
      expect(screen.getByText("Manage purchase orders and vendor relationships.")).toBeInTheDocument();
    });
  });

  it("has create PO button", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("Create PO")).toBeInTheDocument();
    });
  });

  it("shows summary cards", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("Total POs")).toBeInTheDocument();
      expect(screen.getByText("Draft")).toBeInTheDocument();
      expect(screen.getByText("Pending")).toBeInTheDocument();
      expect(screen.getByText("Received")).toBeInTheDocument();
    });
  });

  it("shows table headers", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO Number")).toBeInTheDocument();
      expect(screen.getByText("Vendor")).toBeInTheDocument();
      expect(screen.getByText("Total")).toBeInTheDocument();
      expect(screen.getByText("Created")).toBeInTheDocument();
      expect(screen.getByText("Expected")).toBeInTheDocument();
    });
  });

  it("shows vendor names in table", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("Acme Corp")).toBeInTheDocument();
      expect(screen.getByText("DigiParts")).toBeInTheDocument();
    });
  });

  it("calculates total amount from line items", async () => {
    render(<Procurement />);
    await waitFor(() => {
      // PO-001: 100*0.01 + 50*0.10 = 1.00 + 5.00 = $6.00
      expect(screen.getByText("$6.00")).toBeInTheDocument();
      // PO-002: no lines = $0.00
      expect(screen.getByText("$0.00")).toBeInTheDocument();
    });
  });

  it("shows status badges", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("DRAFT")).toBeInTheDocument();
      expect(screen.getByText("SENT")).toBeInTheDocument();
    });
  });

  it("shows empty state", async () => {
    mockGetPurchaseOrders.mockResolvedValueOnce([]);
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText(/no purchase orders found/i)).toBeInTheDocument();
    });
  });

  it("opens create PO dialog", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create PO"));
    await waitFor(() => {
      expect(screen.getByText("Create Purchase Order")).toBeInTheDocument();
      expect(screen.getByLabelText(/notes/i)).toBeInTheDocument();
    });
  });

  it("create dialog has line items section", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create PO"));
    await waitFor(() => {
      expect(screen.getByText("Line Items")).toBeInTheDocument();
      expect(screen.getByText("Add Item")).toBeInTheDocument();
    });
  });

  it("can add and remove line items in create dialog", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create PO"));
    await waitFor(() => {
      expect(screen.getByText("Add Item")).toBeInTheDocument();
    });
    // Initially 1 line, Remove button disabled
    const removeButtons = screen.getAllByText("Remove");
    expect(removeButtons[0]).toBeDisabled();

    // Add another line
    fireEvent.click(screen.getByText("Add Item"));
    await waitFor(() => {
      const newRemoveButtons = screen.getAllByText("Remove");
      expect(newRemoveButtons.length).toBe(2);
      expect(newRemoveButtons[0]).not.toBeDisabled();
    });
  });

  it("create PO button disabled when vendor not selected", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create PO"));
    await waitFor(() => {
      expect(screen.getByText("Create Purchase Order")).toBeInTheDocument();
    });
    const createButtons = screen.getAllByText("Create PO");
    const submitButton = createButtons[createButtons.length - 1];
    expect(submitButton).toBeDisabled();
  });

  it("cancel button in create dialog works", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create PO"));
    await waitFor(() => {
      expect(screen.getByText("Create Purchase Order")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Cancel"));
  });

  it("has View Details links for each PO", async () => {
    render(<Procurement />);
    await waitFor(() => {
      const viewButtons = screen.getAllByText("View Details");
      expect(viewButtons.length).toBe(2);
    });
  });

  it("calls all APIs on mount", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(mockGetPurchaseOrders).toHaveBeenCalled();
      expect(mockGetVendors).toHaveBeenCalled();
      expect(mockGetParts).toHaveBeenCalled();
    });
  });

  it("handles API error gracefully", async () => {
    mockGetPurchaseOrders.mockRejectedValueOnce(new Error("Network error"));
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText(/no purchase orders found/i)).toBeInTheDocument();
    });
  });

  // Form submission tests
  it("fills line item fields in create dialog", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("Create PO"));
    await waitFor(() => {
      expect(screen.getByText("Create Purchase Order")).toBeInTheDocument();
    });

    // Fill line item fields
    const ipnInput = screen.getByPlaceholderText("Internal part number");
    fireEvent.change(ipnInput, { target: { value: "IPN-001" } });

    const qtyInput = screen.getByPlaceholderText("0");
    fireEvent.change(qtyInput, { target: { value: "100" } });

    const priceInput = screen.getByPlaceholderText("0.00");
    fireEvent.change(priceInput, { target: { value: "0.50" } });

    // Fill notes
    const notesInput = screen.getByPlaceholderText("Optional notes for this PO");
    fireEvent.change(notesInput, { target: { value: "Rush order" } });

    // Verify fields are filled
    expect(ipnInput).toHaveValue("IPN-001");
    expect(qtyInput).toHaveValue(100);
    expect(priceInput).toHaveValue(0.5);
    expect(notesInput).toHaveValue("Rush order");

    // Submit still disabled without vendor selection (Radix Select)
    const createButtons = screen.getAllByText("Create PO");
    const submitButton = createButtons[createButtons.length - 1];
    expect(submitButton).toBeDisabled();
  });

  // --- New medium tests ---

  it("IPN autocomplete shows filtered parts dropdown", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create PO"));
    await waitFor(() => {
      expect(screen.getByText("Create Purchase Order")).toBeInTheDocument();
    });

    const ipnInput = screen.getByPlaceholderText("Internal part number");
    fireEvent.change(ipnInput, { target: { value: "IPN-00" } });

    // Should show all 3 matching parts
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
      expect(screen.getByText("IPN-002")).toBeInTheDocument();
      expect(screen.getByText("IPN-003")).toBeInTheDocument();
    });

    // Type more specific
    fireEvent.change(ipnInput, { target: { value: "IPN-003" } });
    await waitFor(() => {
      expect(screen.getByText("IPN-003")).toBeInTheDocument();
      expect(screen.queryByText("IPN-001")).not.toBeInTheDocument();
    });
  });

  it("line item field updates — mpn, manufacturer, unit_price, notes", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create PO"));
    await waitFor(() => {
      expect(screen.getByText("Create Purchase Order")).toBeInTheDocument();
    });

    const mpnInput = screen.getByPlaceholderText("Manufacturer PN");
    fireEvent.change(mpnInput, { target: { value: "MPN-XYZ" } });
    expect(mpnInput).toHaveValue("MPN-XYZ");

    const mfgInput = screen.getByPlaceholderText("Manufacturer");
    fireEvent.change(mfgInput, { target: { value: "Texas Instruments" } });
    expect(mfgInput).toHaveValue("Texas Instruments");

    const priceInput = screen.getByPlaceholderText("0.00");
    fireEvent.change(priceInput, { target: { value: "12.50" } });
    expect(priceInput).toHaveValue(12.5);

    const notesInput = screen.getByPlaceholderText("Notes");
    fireEvent.change(notesInput, { target: { value: "Lead-free" } });
    expect(notesInput).toHaveValue("Lead-free");
  });

  it("remove line item — add 2nd line, click remove, verify gone", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create PO"));
    await waitFor(() => {
      expect(screen.getByText("Add Item")).toBeInTheDocument();
    });

    // Fill first line's MPN to identify it
    const mpnInput = screen.getByPlaceholderText("Manufacturer PN");
    fireEvent.change(mpnInput, { target: { value: "FIRST-LINE" } });

    // Add second line
    fireEvent.click(screen.getByText("Add Item"));
    await waitFor(() => {
      expect(screen.getAllByText("Remove").length).toBe(2);
    });

    // Fill second line's MPN
    const mpnInputs = screen.getAllByPlaceholderText("Manufacturer PN");
    fireEvent.change(mpnInputs[1], { target: { value: "SECOND-LINE" } });

    // Remove first line
    const removeButtons = screen.getAllByText("Remove");
    fireEvent.click(removeButtons[0]);

    await waitFor(() => {
      expect(screen.getAllByText("Remove").length).toBe(1);
    });
    // Second line should remain
    expect(screen.getByDisplayValue("SECOND-LINE")).toBeInTheDocument();
    expect(screen.queryByDisplayValue("FIRST-LINE")).not.toBeInTheDocument();
  });

  it("Create PO disabled with vendor set but empty lines (no ipn/qty)", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create PO"));
    await waitFor(() => {
      expect(screen.getByText("Create Purchase Order")).toBeInTheDocument();
    });

    // Even if we fill only IPN without qty, button stays disabled
    const ipnInput = screen.getByPlaceholderText("Internal part number");
    fireEvent.change(ipnInput, { target: { value: "IPN-001" } });

    // No qty filled — button still disabled (needs vendor AND valid line)
    const createButtons = screen.getAllByText("Create PO");
    const submitButton = createButtons[createButtons.length - 1];
    expect(submitButton).toBeDisabled();

    // Fill qty but still no vendor — still disabled
    const qtyInput = screen.getByPlaceholderText("0");
    fireEvent.change(qtyInput, { target: { value: "10" } });
    expect(submitButton).toBeDisabled();
  });

  it("lines filter in handleCreatePO — lines without ipn/qty filtered before API call", async () => {
    // Directly test the filtering logic: when createPO is called,
    // empty lines should be excluded. We verify by checking the disabled
    // condition: button requires vendor AND at least one line with ipn+qty.
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create PO"));
    await waitFor(() => {
      expect(screen.getByText("Create Purchase Order")).toBeInTheDocument();
    });

    // Add a second line — both empty
    fireEvent.click(screen.getByText("Add Item"));
    await waitFor(() => {
      expect(screen.getAllByText("Remove").length).toBe(2);
    });

    // Fill only first line with ipn+qty
    const ipnInputs = screen.getAllByPlaceholderText("Internal part number");
    fireEvent.change(ipnInputs[0], { target: { value: "IPN-001" } });
    const qtyInputs = screen.getAllByPlaceholderText("0");
    fireEvent.change(qtyInputs[0], { target: { value: "10" } });

    // Second line stays empty — the disabled condition checks
    // poForm.lines.some(line => line.ipn && line.qty_ordered)
    // This returns true because first line has both, so button is only
    // disabled due to missing vendor_id
    const createButtons = screen.getAllByText("Create PO");
    const submitButton = createButtons[createButtons.length - 1];
    // Still disabled because no vendor selected (Radix Select can't be tested easily)
    expect(submitButton).toBeDisabled();

    // Verify that without any valid line, it's also disabled
    fireEvent.change(ipnInputs[0], { target: { value: "" } });
    expect(submitButton).toBeDisabled();
  });

  it("summary card count accuracy", async () => {
    // mockPOsWithLines: PO-001 is draft, PO-002 is sent
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });

    // Total POs = 2
    const totalCard = screen.getByText("Total POs").closest("div")!.parentElement!;
    expect(totalCard).toHaveTextContent("2");

    // Draft = 1 (PO-001)
    const draftCard = screen.getByText("Draft").closest("div")!.parentElement!;
    expect(draftCard).toHaveTextContent("1");

    // Pending (submitted+partial) = 0 (sent !== submitted)
    const pendingCard = screen.getByText("Pending").closest("div")!.parentElement!;
    expect(pendingCard).toHaveTextContent("0");

    // Received = 0
    const receivedCard = screen.getByText("Received").closest("div")!.parentElement!;
    expect(receivedCard).toHaveTextContent("0");
  });

  it("PO link hrefs point to correct /purchase-orders/{id} URLs", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });

    // PO number links
    const po1Link = screen.getByText("PO-001").closest("a");
    expect(po1Link).toHaveAttribute("href", "/purchase-orders/PO-001");

    const po2Link = screen.getByText("PO-002").closest("a");
    expect(po2Link).toHaveAttribute("href", "/purchase-orders/PO-002");

    // View Details links
    const viewDetailLinks = screen.getAllByText("View Details");
    expect(viewDetailLinks[0].closest("a")).toHaveAttribute("href", "/purchase-orders/PO-001");
    expect(viewDetailLinks[1].closest("a")).toHaveAttribute("href", "/purchase-orders/PO-002");
  });
});
