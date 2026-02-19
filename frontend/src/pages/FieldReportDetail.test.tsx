import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";

const mockReport = {
  id: "FR-001",
  title: "Overheating Issue",
  report_type: "failure",
  status: "open",
  priority: "high",
  description: "Device overheating in field",
  device_serial: "SN-1234",
  reporter: "John Doe",
  created_at: "2024-01-15",
  updated_at: "2024-01-15",
  linked_ncr_id: null,
};

const mockGetFieldReport = vi.fn().mockResolvedValue(mockReport);
const mockUpdateFieldReport = vi.fn().mockResolvedValue(mockReport);
const mockCreateNCRFromFieldReport = vi.fn().mockResolvedValue({ id: "NCR-001", title: "From FR-001" });

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useParams: () => ({ id: "FR-001" }), useNavigate: () => mockNavigate };
});

vi.mock("../lib/api", () => ({
  api: {
    getFieldReport: (...args: any[]) => mockGetFieldReport(...args),
    updateFieldReport: (...args: any[]) => mockUpdateFieldReport(...args),
    createNCRFromFieldReport: (...args: any[]) => mockCreateNCRFromFieldReport(...args),
  },
}));

import FieldReportDetail from "./FieldReportDetail";

beforeEach(() => vi.clearAllMocks());

describe("FieldReportDetail", () => {
  it("renders loading state initially", () => {
    mockGetFieldReport.mockReturnValue(new Promise(() => {}));
    render(<FieldReportDetail />);
    expect(document.querySelector(".animate-spin")).toBeInTheDocument();
  });

  it("renders report detail after loading", async () => {
    render(<FieldReportDetail />);
    await waitFor(() => {
      expect(screen.getByText(/Overheating Issue/)).toBeInTheDocument();
    });
  });

  it("shows report type and status badges", async () => {
    render(<FieldReportDetail />);
    await waitFor(() => {
      expect(screen.getByText("failure")).toBeInTheDocument();
      expect(screen.getByText("open")).toBeInTheDocument();
      expect(screen.getByText("high")).toBeInTheDocument();
    });
  });

  it("shows investigate button for open reports", async () => {
    render(<FieldReportDetail />);
    await waitFor(() => {
      expect(screen.getByText("Investigate")).toBeInTheDocument();
    });
  });

  it("calls updateFieldReport when changing status", async () => {
    render(<FieldReportDetail />);
    await waitFor(() => screen.getByText("Investigate"));
    fireEvent.click(screen.getByText("Investigate"));
    await waitFor(() => {
      expect(mockUpdateFieldReport).toHaveBeenCalledWith("FR-001", { status: "investigating" });
    });
  });

  it("navigates back on error fetching report", async () => {
    mockGetFieldReport.mockRejectedValueOnce(new Error("Not found"));
    render(<FieldReportDetail />);
    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith("/field-reports");
    });
  });

  it("fetches report with correct id", async () => {
    render(<FieldReportDetail />);
    await waitFor(() => {
      expect(mockGetFieldReport).toHaveBeenCalledWith("FR-001");
    });
  });
});
