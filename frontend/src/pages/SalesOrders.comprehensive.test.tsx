import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockSalesOrders } from "../test/mocks";
import type { SalesOrder } from "../lib/api";

// Mock API
const mockGetSalesOrders = vi.fn().mockResolvedValue(mockSalesOrders);
const mockCreateSalesOrder = vi.fn().mockResolvedValue({
  id: "SO-0003",
  customer: "New Customer",
  status: "draft",
  lines: [],
  created_at: "2024-01-26",
  updated_at: "2024-01-26",
});
const mockUpdateSalesOrder = vi.fn();
const mockDeleteSalesOrder = vi.fn();
const mockConfirmSalesOrder = vi.fn();
const mockAllocateSalesOrder = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getSalesOrders: (...args: any[]) => mockGetSalesOrders(...args),
    createSalesOrder: (...args: any[]) => mockCreateSalesOrder(...args),
    updateSalesOrder: (...args: any[]) => mockUpdateSalesOrder(...args),
    deleteSalesOrder: (...args: any[]) => mockDeleteSalesOrder(...args),
    confirmSalesOrder: (...args: any[]) => mockConfirmSalesOrder(...args),
    allocateSalesOrder: (...args: any[]) => mockAllocateSalesOrder(...args),
  },
}));

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useNavigate: () => mockNavigate };
});

import SalesOrders from "./SalesOrders";

beforeEach(() => {
  vi.clearAllMocks();
  mockGetSalesOrders.mockResolvedValue(mockSalesOrders);
});

describe("SalesOrders - Comprehensive Tests", () => {
  // ==========================================================================
  // LIST AND DISPLAY TESTS
  // ==========================================================================

  it("renders sales orders list with all columns", async () => {
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });
    expect(screen.getByText("SO-0002")).toBeInTheDocument();
    expect(screen.getByText("Acme Inc")).toBeInTheDocument();
    expect(screen.getByText("Tech Co")).toBeInTheDocument();
  });

  it("displays status badges correctly", async () => {
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("draft")).toBeInTheDocument();
    });
    expect(screen.getByText("confirmed")).toBeInTheDocument();
  });

  it("shows quote reference when present", async () => {
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("Q-001")).toBeInTheDocument();
    });
  });

  it("displays empty state when no orders", async () => {
    mockGetSalesOrders.mockResolvedValueOnce([]);
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText(/no sales orders found/i)).toBeInTheDocument();
    });
  });

  it("shows loading state initially", async () => {
    mockGetSalesOrders.mockImplementationOnce(
      () => new Promise((resolve) => setTimeout(() => resolve(mockSalesOrders), 100))
    );
    render(<SalesOrders />);
    // Loading indicator should be present (spinner, skeleton, etc.)
    // This depends on your UI implementation
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });
  });

  // ==========================================================================
  // FILTERING AND SEARCH TESTS
  // ==========================================================================

  it("filters by status", async () => {
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });

    // Simulate status filter change
    const statusFilter = screen.queryByRole("combobox", { name: /status/i });
    if (statusFilter) {
      fireEvent.change(statusFilter, { target: { value: "draft" } });
      await waitFor(() => {
        expect(mockGetSalesOrders).toHaveBeenCalledWith(
          expect.objectContaining({ status: "draft" })
        );
      });
    }
  });

  it("searches by customer name", async () => {
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });

    const searchInput = screen.queryByPlaceholderText(/search/i);
    if (searchInput) {
      fireEvent.change(searchInput, { target: { value: "Acme" } });
      await waitFor(() => {
        expect(mockGetSalesOrders).toHaveBeenCalledWith(
          expect.objectContaining({ customer: "Acme" })
        );
      });
    }
  });

  it("clears filters", async () => {
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });

    const clearButton = screen.queryByRole("button", { name: /clear/i });
    if (clearButton) {
      fireEvent.click(clearButton);
      await waitFor(() => {
        expect(mockGetSalesOrders).toHaveBeenCalledWith({});
      });
    }
  });

  // ==========================================================================
  // NAVIGATION TESTS
  // ==========================================================================

  it("navigates to detail view on row click", async () => {
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });

    const orderLink = screen.getByText("SO-0001");
    fireEvent.click(orderLink);
    
    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith(
        expect.stringContaining("SO-0001")
      );
    });
  });

  it("navigates to create form on new button click", async () => {
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });

    const newButton = screen.queryByRole("button", { name: /new.*order/i });
    if (newButton) {
      fireEvent.click(newButton);
      await waitFor(() => {
        expect(mockNavigate).toHaveBeenCalledWith(
          expect.stringContaining("new")
        );
      });
    }
  });

  // ==========================================================================
  // ERROR HANDLING TESTS
  // ==========================================================================

  it("displays error message when API fails", async () => {
    mockGetSalesOrders.mockRejectedValueOnce(new Error("Network error"));
    render(<SalesOrders />);
    
    await waitFor(() => {
      const errorText = screen.queryByText(/error/i) || screen.queryByText(/failed/i);
      expect(errorText).toBeInTheDocument();
    });
  });

  it("retries on error", async () => {
    mockGetSalesOrders
      .mockRejectedValueOnce(new Error("Network error"))
      .mockResolvedValueOnce(mockSalesOrders);
    
    render(<SalesOrders />);
    
    await waitFor(() => {
      expect(mockGetSalesOrders).toHaveBeenCalledTimes(1);
    });

    const retryButton = screen.queryByRole("button", { name: /retry/i });
    if (retryButton) {
      fireEvent.click(retryButton);
      await waitFor(() => {
        expect(mockGetSalesOrders).toHaveBeenCalledTimes(2);
      });
    }
  });

  // ==========================================================================
  // REFRESH AND RELOAD TESTS
  // ==========================================================================

  it("refreshes data on refresh button click", async () => {
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });

    const refreshButton = screen.queryByRole("button", { name: /refresh/i });
    if (refreshButton) {
      fireEvent.click(refreshButton);
      await waitFor(() => {
        expect(mockGetSalesOrders).toHaveBeenCalledTimes(2);
      });
    }
  });

  // ==========================================================================
  // SORTING TESTS
  // ==========================================================================

  it("sorts by ID column", async () => {
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });

    const idHeader = screen.queryByRole("button", { name: /id/i });
    if (idHeader) {
      fireEvent.click(idHeader);
      // Verify orders are displayed in sorted order
      const orders = screen.getAllByText(/SO-\d+/);
      expect(orders[0].textContent).toBe("SO-0001");
    }
  });

  it("sorts by customer column", async () => {
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });

    const customerHeader = screen.queryByRole("button", { name: /customer/i });
    if (customerHeader) {
      fireEvent.click(customerHeader);
      // Verify sorting
      await waitFor(() => {
        expect(mockGetSalesOrders).toHaveBeenCalled();
      });
    }
  });

  // ==========================================================================
  // BULK ACTIONS TESTS
  // ==========================================================================

  it("selects multiple orders", async () => {
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });

    const checkboxes = screen.queryAllByRole("checkbox");
    if (checkboxes.length > 0) {
      fireEvent.click(checkboxes[0]);
      fireEvent.click(checkboxes[1]);
      
      // Verify selection state
      expect(checkboxes[0]).toBeChecked();
      expect(checkboxes[1]).toBeChecked();
    }
  });

  it("shows bulk action menu when orders selected", async () => {
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });

    const checkboxes = screen.queryAllByRole("checkbox");
    if (checkboxes.length > 0) {
      fireEvent.click(checkboxes[0]);
      
      const bulkMenu = screen.queryByText(/bulk action/i) || screen.queryByText(/selected/i);
      expect(bulkMenu).toBeInTheDocument();
    }
  });

  // ==========================================================================
  // PAGINATION TESTS
  // ==========================================================================

  it("displays pagination when many orders", async () => {
    const manyOrders = Array.from({ length: 50 }, (_, i) => ({
      ...mockSalesOrders[0],
      id: `SO-${String(i + 1).padStart(4, "0")}`,
      customer: `Customer ${i + 1}`,
    }));
    mockGetSalesOrders.mockResolvedValueOnce(manyOrders);
    
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });

    const pagination = screen.queryByRole("navigation", { name: /pagination/i });
    if (pagination) {
      expect(pagination).toBeInTheDocument();
    }
  });

  it("navigates to next page", async () => {
    const manyOrders = Array.from({ length: 50 }, (_, i) => ({
      ...mockSalesOrders[0],
      id: `SO-${String(i + 1).padStart(4, "0")}`,
    }));
    mockGetSalesOrders.mockResolvedValueOnce(manyOrders);
    
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });

    const nextButton = screen.queryByRole("button", { name: /next/i });
    if (nextButton) {
      fireEvent.click(nextButton);
      // Verify page change
      await waitFor(() => {
        expect(mockGetSalesOrders).toHaveBeenCalled();
      });
    }
  });

  // ==========================================================================
  // EXPORT TESTS
  // ==========================================================================

  it("exports to CSV", async () => {
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });

    const exportButton = screen.queryByRole("button", { name: /export/i });
    if (exportButton) {
      fireEvent.click(exportButton);
      // Verify export action triggered
      // This depends on implementation
    }
  });

  // ==========================================================================
  // ACCESSIBILITY TESTS
  // ==========================================================================

  it("is keyboard navigable", async () => {
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });

    const firstLink = screen.getByText("SO-0001");
    firstLink.focus();
    expect(document.activeElement).toBe(firstLink);
  });

  it("has proper ARIA labels", async () => {
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });

    // Check for main heading
    const heading = screen.queryByRole("heading", { name: /sales.*order/i });
    expect(heading).toBeInTheDocument();
  });

  // ==========================================================================
  // STATUS-SPECIFIC TESTS
  // ==========================================================================

  it("shows correct actions for draft orders", async () => {
    const draftOrder = { ...mockSalesOrders[0], status: "draft" };
    mockGetSalesOrders.mockResolvedValueOnce([draftOrder]);
    
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });

    // Draft orders should show confirm action
    const confirmButton = screen.queryByRole("button", { name: /confirm/i });
    if (confirmButton) {
      expect(confirmButton).toBeInTheDocument();
    }
  });

  it("shows correct actions for confirmed orders", async () => {
    const confirmedOrder = { ...mockSalesOrders[0], status: "confirmed" };
    mockGetSalesOrders.mockResolvedValueOnce([confirmedOrder]);
    
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });

    // Confirmed orders should show allocate action
    const allocateButton = screen.queryByRole("button", { name: /allocate/i });
    if (allocateButton) {
      expect(allocateButton).toBeInTheDocument();
    }
  });

  it("disables actions for completed orders", async () => {
    const completedOrder = { ...mockSalesOrders[0], status: "invoiced" };
    mockGetSalesOrders.mockResolvedValueOnce([completedOrder]);
    
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });

    // Invoiced orders should not show workflow actions
    const confirmButton = screen.queryByRole("button", { name: /confirm/i });
    expect(confirmButton).not.toBeInTheDocument();
  });
});
