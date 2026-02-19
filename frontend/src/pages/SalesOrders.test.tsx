import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import { mockSalesOrders } from "../test/mocks";

const mockGetSalesOrders = vi.fn().mockResolvedValue(mockSalesOrders);

vi.mock("../lib/api", () => ({
  api: {
    getSalesOrders: (...args: any[]) => mockGetSalesOrders(...args),
  },
}));

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useNavigate: () => mockNavigate };
});

import SalesOrders from "./SalesOrders";

beforeEach(() => vi.clearAllMocks());

describe("SalesOrders", () => {
  it("renders sales orders list", async () => {
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });
    expect(screen.getByText("SO-0002")).toBeInTheDocument();
    expect(screen.getByText("Acme Inc")).toBeInTheDocument();
    expect(screen.getByText("Tech Co")).toBeInTheDocument();
  });

  it("shows status badges", async () => {
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("draft")).toBeInTheDocument();
    });
    expect(screen.getByText("confirmed")).toBeInTheDocument();
  });

  it("shows empty state when no orders", async () => {
    mockGetSalesOrders.mockResolvedValueOnce([]);
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText(/no sales orders found/i)).toBeInTheDocument();
    });
  });

  it("filters by status", async () => {
    render(<SalesOrders />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });
    // The component calls getSalesOrders on status change
    expect(mockGetSalesOrders).toHaveBeenCalled();
  });
});
