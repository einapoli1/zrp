import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";

const mockGetRFQ = vi.fn();
const mockUpdateRFQ = vi.fn();
const mockSendRFQ = vi.fn();
const mockCloseRFQ = vi.fn();
const mockAwardRFQ = vi.fn();
const mockCompareRFQ = vi.fn();
const mockCreateRFQQuote = vi.fn();

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useParams: () => ({ id: "RFQ-2026-0001" }), useNavigate: () => vi.fn() };
});

vi.mock("../lib/api", () => ({
  api: {
    getRFQ: (...args: any[]) => mockGetRFQ(...args),
    updateRFQ: (...args: any[]) => mockUpdateRFQ(...args),
    sendRFQ: (...args: any[]) => mockSendRFQ(...args),
    closeRFQ: (...args: any[]) => mockCloseRFQ(...args),
    awardRFQ: (...args: any[]) => mockAwardRFQ(...args),
    compareRFQ: (...args: any[]) => mockCompareRFQ(...args),
    createRFQQuote: (...args: any[]) => mockCreateRFQQuote(...args),
  },
}));

import RFQDetail from "./RFQDetail";

const mockRFQ = {
  id: "RFQ-2026-0001", title: "Resistor Bulk Quote", status: "draft",
  created_by: "admin", created_at: "2026-01-15", updated_at: "2026-01-15",
  due_date: "2026-02-01", notes: "Need pricing for Q2",
  lines: [], vendors: [],
};

beforeEach(() => {
  vi.clearAllMocks();
  mockGetRFQ.mockResolvedValue(mockRFQ);
  mockCompareRFQ.mockResolvedValue({ lines: [], vendors: [], matrix: {} });
});

describe("RFQDetail", () => {
  it("renders loading state initially", () => {
    mockGetRFQ.mockReturnValue(new Promise(() => {}));
    render(<RFQDetail />);
    expect(screen.getByText(/loading/i)).toBeInTheDocument();
  });

  it("renders RFQ detail after loading", async () => {
    render(<RFQDetail />);
    await waitFor(() => {
      expect(screen.getByText("Resistor Bulk Quote")).toBeInTheDocument();
    });
  });

  it("shows RFQ status badge", async () => {
    render(<RFQDetail />);
    await waitFor(() => {
      expect(screen.getByText("draft")).toBeInTheDocument();
    });
  });

  it("calls getRFQ with correct id", async () => {
    render(<RFQDetail />);
    await waitFor(() => {
      expect(mockGetRFQ).toHaveBeenCalledWith("RFQ-2026-0001");
    });
  });

  it("calls compareRFQ on load", async () => {
    render(<RFQDetail />);
    await waitFor(() => {
      expect(mockCompareRFQ).toHaveBeenCalledWith("RFQ-2026-0001");
    });
  });
});
