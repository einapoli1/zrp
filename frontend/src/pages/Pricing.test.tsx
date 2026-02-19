import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import type { ProductPricing, CostAnalysis } from "../lib/api";

const mockGetProductPricing = vi.fn();
const mockGetCostAnalysis = vi.fn();
const mockCreateProductPricing = vi.fn();
const mockUpdateProductPricing = vi.fn();
const mockDeleteProductPricing = vi.fn();
const mockGetProductPricingHistory = vi.fn();
const mockBulkUpdateProductPricing = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getProductPricing: (...args: any[]) => mockGetProductPricing(...args),
    getCostAnalysis: (...args: any[]) => mockGetCostAnalysis(...args),
    createProductPricing: (...args: any[]) => mockCreateProductPricing(...args),
    updateProductPricing: (...args: any[]) => mockUpdateProductPricing(...args),
    deleteProductPricing: (...args: any[]) => mockDeleteProductPricing(...args),
    getProductPricingHistory: (...args: any[]) => mockGetProductPricingHistory(...args),
    bulkUpdateProductPricing: (...args: any[]) => mockBulkUpdateProductPricing(...args),
  },
}));

import Pricing from "./Pricing";

const mockPricingData: ProductPricing[] = [
  {
    id: 1, product_ipn: "IPN-001", pricing_tier: "standard", min_qty: 1, max_qty: 100,
    unit_price: 15.00, currency: "USD", effective_date: "2024-01-01", created_at: "2024-01-01", updated_at: "2024-01-01",
  },
  {
    id: 2, product_ipn: "IPN-001", pricing_tier: "volume", min_qty: 100, max_qty: 1000,
    unit_price: 12.00, currency: "USD", effective_date: "2024-01-01", created_at: "2024-01-01", updated_at: "2024-01-01",
  },
  {
    id: 3, product_ipn: "IPN-002", pricing_tier: "standard", min_qty: 1, max_qty: 100,
    unit_price: 5.00, currency: "USD", effective_date: "2024-01-01", created_at: "2024-01-01", updated_at: "2024-01-01",
  },
];

const mockAnalysisData: CostAnalysis[] = [
  {
    id: 1, product_ipn: "IPN-001", bom_cost: 5.00, labor_cost: 2.00, overhead_cost: 1.00,
    total_cost: 8.00, margin_pct: 46.67, selling_price: 15.00, last_calculated: "2024-01-01", created_at: "2024-01-01",
  },
  {
    id: 2, product_ipn: "IPN-002", bom_cost: 4.00, labor_cost: 0.50, overhead_cost: 0.20,
    total_cost: 4.70, margin_pct: 6.0, selling_price: 5.00, last_calculated: "2024-01-01", created_at: "2024-01-01",
  },
];

beforeEach(() => {
  vi.clearAllMocks();
  mockGetProductPricing.mockResolvedValue(mockPricingData);
  mockGetCostAnalysis.mockResolvedValue(mockAnalysisData);
  mockGetProductPricingHistory.mockResolvedValue(mockPricingData.slice(0, 2));
  mockCreateProductPricing.mockResolvedValue(mockPricingData[0]);
  mockUpdateProductPricing.mockResolvedValue(mockPricingData[0]);
  mockDeleteProductPricing.mockResolvedValue({ status: "deleted" });
  mockBulkUpdateProductPricing.mockResolvedValue({ updated: 2, total: 2 });
});

const waitForLoad = () => waitFor(() => expect(screen.getAllByText("IPN-001").length).toBeGreaterThan(0));

describe("Pricing", () => {
  it("renders loading state", () => {
    render(<Pricing />);
    expect(screen.getByText("Loading pricing data...")).toBeInTheDocument();
  });

  it("renders page title", async () => {
    render(<Pricing />);
    await waitForLoad();
    expect(screen.getByText("Pricing")).toBeInTheDocument();
  });

  it("renders pricing table with product data", async () => {
    render(<Pricing />);
    await waitForLoad();
    expect(screen.getAllByText("IPN-001").length).toBeGreaterThan(0);
    expect(screen.getByText("IPN-002")).toBeInTheDocument();
  });

  it("shows margin with color coding - green for >30%", async () => {
    render(<Pricing />);
    await waitForLoad();
    // IPN-001 has 46.67% margin - should be green
    const marginCells = screen.getAllByText(/46\.7%/);
    expect(marginCells.length).toBeGreaterThan(0);
  });

  it("shows margin with color coding - red for <15%", async () => {
    render(<Pricing />);
    await waitForLoad();
    // IPN-002 has 6% margin - should be red
    const marginCells = screen.getAllByText(/6\.0%/);
    expect(marginCells.length).toBeGreaterThan(0);
  });

  it("has add pricing button", async () => {
    render(<Pricing />);
    await waitForLoad();
    expect(screen.getByText("Add Pricing")).toBeInTheDocument();
  });

  it("has cost analysis tab", async () => {
    render(<Pricing />);
    await waitForLoad();
    expect(screen.getByRole("tab", { name: /cost analysis/i })).toBeInTheDocument();
  });

  it("calls API on mount", async () => {
    render(<Pricing />);
    await waitForLoad();
    expect(mockGetProductPricing).toHaveBeenCalled();
    expect(mockGetCostAnalysis).toHaveBeenCalled();
  });
});
