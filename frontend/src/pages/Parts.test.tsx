import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockParts, mockCategories } from "../test/mocks";

const mockGetParts = vi.fn();
const mockGetCategories = vi.fn();
const mockCreatePart = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getParts: (...args: any[]) => mockGetParts(...args),
    getCategories: (...args: any[]) => mockGetCategories(...args),
    createPart: (...args: any[]) => mockCreatePart(...args),
    deletePart: vi.fn().mockResolvedValue(undefined),
  },
}));

import Parts from "./Parts";

beforeEach(() => {
  vi.clearAllMocks();
  mockGetParts.mockResolvedValue({ data: mockParts, meta: { total: 3, page: 1, limit: 50 } });
  mockGetCategories.mockResolvedValue(mockCategories);
  mockCreatePart.mockResolvedValue(mockParts[0]);
});

// Helper: wait for parts to load
const waitForLoad = () => waitFor(() => expect(screen.getByText("IPN-001")).toBeInTheDocument());

describe("Parts", () => {
  it("renders page title and subtitle", async () => {
    render(<Parts />);
    await waitForLoad();
    expect(screen.getByText("Parts")).toBeInTheDocument();
    expect(screen.getByText("Manage your parts inventory and specifications")).toBeInTheDocument();
  });

  it("renders parts table after loading", async () => {
    render(<Parts />);
    await waitForLoad();
    expect(screen.getByText("IPN-002")).toBeInTheDocument();
    expect(screen.getByText("IPN-003")).toBeInTheDocument();
  });

  it("shows loading skeletons initially", () => {
    mockGetParts.mockReturnValue(new Promise(() => {}));
    render(<Parts />);
    expect(screen.queryByText("IPN-001")).not.toBeInTheDocument();
  });

  it("has search input with placeholder", async () => {
    render(<Parts />);
    await waitForLoad();
    expect(screen.getByPlaceholderText(/search parts by ipn/i)).toBeInTheDocument();
  });

  it("has add part button", async () => {
    render(<Parts />);
    await waitForLoad();
    expect(screen.getByText("Add Part")).toBeInTheDocument();
  });

  it("opens create dialog on button click", async () => {
    render(<Parts />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Add Part"));
    await waitFor(() => {
      expect(screen.getByText("Add New Part")).toBeInTheDocument();
      expect(screen.getByText("Create a new part in your inventory system.")).toBeInTheDocument();
    });
  });

  it("shows parts count", async () => {
    render(<Parts />);
    await waitForLoad();
    // Text nodes split: find container with "Parts (3)"
    const el = screen.getByText((_, element) => {
      if (!element || element.tagName === 'H1') return false;
      const text = element.textContent || '';
      return text.includes('Parts (') && text.includes('3') && element.classList?.contains('font-semibold');
    });
    expect(el).toBeInTheDocument();
  });

  it("shows empty state when no parts", async () => {
    mockGetParts.mockResolvedValue({ data: [], meta: { total: 0, page: 1, limit: 50 } });
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText(/no parts available/i)).toBeInTheDocument();
    });
  });

  it("calls getParts and getCategories on mount", async () => {
    render(<Parts />);
    await waitForLoad();
    expect(mockGetParts).toHaveBeenCalled();
    expect(mockGetCategories).toHaveBeenCalled();
  });

  it("handles API error gracefully", async () => {
    mockGetParts.mockRejectedValue(new Error("fail"));
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText("Parts")).toBeInTheDocument();
    });
  });

  it("renders Filters card", async () => {
    render(<Parts />);
    await waitForLoad();
    expect(screen.getByText("Filters")).toBeInTheDocument();
  });

  it("shows table headers", async () => {
    render(<Parts />);
    await waitForLoad();
    expect(screen.getByText("IPN")).toBeInTheDocument();
    expect(screen.getByText("Category")).toBeInTheDocument();
    expect(screen.getByText("Description")).toBeInTheDocument();
    expect(screen.getByText("Cost")).toBeInTheDocument();
    expect(screen.getByText("Stock")).toBeInTheDocument();
    expect(screen.getByText("Status")).toBeInTheDocument();
  });

  it("create dialog has required form fields", async () => {
    render(<Parts />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Add Part"));
    await waitFor(() => {
      expect(screen.getByLabelText("IPN *")).toBeInTheDocument();
      expect(screen.getByLabelText("Description")).toBeInTheDocument();
      expect(screen.getByLabelText("Cost ($)")).toBeInTheDocument();
      expect(screen.getByLabelText("Price ($)")).toBeInTheDocument();
      expect(screen.getByLabelText("Minimum Stock")).toBeInTheDocument();
      expect(screen.getByLabelText("Current Stock")).toBeInTheDocument();
      expect(screen.getByLabelText("Location")).toBeInTheDocument();
      expect(screen.getByLabelText("Vendor")).toBeInTheDocument();
    });
  });

  it("create button disabled when IPN empty", async () => {
    render(<Parts />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Add Part"));
    await waitFor(() => {
      expect(screen.getByText("Create Part")).toBeDisabled();
    });
  });

  it("create button enabled when IPN filled", async () => {
    render(<Parts />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Add Part"));
    await waitFor(() => expect(screen.getByLabelText("IPN *")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("IPN *"), { target: { value: "NEW-001" } });
    expect(screen.getByText("Create Part")).not.toBeDisabled();
  });

  it("submits create form and closes dialog", async () => {
    render(<Parts />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Add Part"));
    await waitFor(() => expect(screen.getByLabelText("IPN *")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("IPN *"), { target: { value: "NEW-001" } });
    fireEvent.change(screen.getByLabelText("Description"), { target: { value: "Test part" } });
    fireEvent.click(screen.getByText("Create Part"));
    await waitFor(() => {
      expect(mockCreatePart).toHaveBeenCalledWith(
        expect.objectContaining({ ipn: "NEW-001", description: "Test part" })
      );
    });
  });

  it("cancel button closes create dialog", async () => {
    render(<Parts />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Add Part"));
    await waitFor(() => expect(screen.getByText("Add New Part")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Cancel"));
    await waitFor(() => {
      expect(screen.queryByText("Add New Part")).not.toBeInTheDocument();
    });
  });

  it("shows page info", async () => {
    render(<Parts />);
    await waitForLoad();
    // Text may be split across elements
    const el = screen.getByText((_, element) => element?.textContent === 'Page 1 of 1' || false);
    expect(el).toBeInTheDocument();
  });

  it("shows pagination when more than one page", async () => {
    mockGetParts.mockResolvedValue({ data: mockParts, meta: { total: 100, page: 1, limit: 50 } });
    render(<Parts />);
    await waitForLoad();
    expect(screen.getByText("Previous")).toBeInTheDocument();
    expect(screen.getByText("Next")).toBeInTheDocument();
    expect(screen.getByText("Previous").closest("button")).toBeDisabled();
    expect(screen.getByText("Next").closest("button")).not.toBeDisabled();
  });

  it("shows showing X to Y of Z text for pagination", async () => {
    mockGetParts.mockResolvedValue({ data: mockParts, meta: { total: 100, page: 1, limit: 50 } });
    render(<Parts />);
    await waitForLoad();
    expect(screen.getByText(/Showing 1 to 50 of 100 parts/)).toBeInTheDocument();
  });

  it("shows filtered empty state message when search active", async () => {
    render(<Parts />);
    await waitForLoad();
    // Change search, triggering a re-fetch
    mockGetParts.mockResolvedValue({ data: [], meta: { total: 0, page: 1, limit: 50 } });
    const searchInput = screen.getByPlaceholderText(/search parts by ipn/i);
    fireEvent.change(searchInput, { target: { value: "nonexistent" } });
    await waitFor(() => {
      expect(screen.getByText("No parts found matching your criteria")).toBeInTheDocument();
    });
  });

  it("handles category error gracefully", async () => {
    mockGetCategories.mockRejectedValue(new Error("fail"));
    render(<Parts />);
    await waitForLoad();
    expect(screen.getByText("Parts")).toBeInTheDocument();
  });

  it("search calls getParts with query param", async () => {
    render(<Parts />);
    await waitForLoad();
    mockGetParts.mockClear();
    const searchInput = screen.getByPlaceholderText(/search parts by ipn/i);
    fireEvent.change(searchInput, { target: { value: "resistor" } });
    await waitFor(() => {
      expect(mockGetParts).toHaveBeenCalledWith(
        expect.objectContaining({ q: "resistor", page: 1 })
      );
    });
  });
});
