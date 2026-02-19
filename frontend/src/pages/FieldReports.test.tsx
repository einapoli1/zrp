import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "../test/test-utils";
import FieldReports from "./FieldReports";

const mockFieldReports = [
  {
    id: "FR-2026-001",
    title: "Motor overheating",
    report_type: "failure",
    status: "open",
    priority: "high",
    customer_name: "Acme Corp",
    site_location: "Plant 3",
    device_ipn: "MOT-001",
    device_serial: "SN-123",
    reported_by: "John",
    reported_at: "2026-01-15",
    description: "Motor runs hot",
    root_cause: "",
    resolution: "",
    ncr_id: "",
    eco_id: "",
    created_at: "2026-01-15",
    updated_at: "2026-01-15",
  },
  {
    id: "FR-2026-002",
    title: "Display flickering",
    report_type: "complaint",
    status: "investigating",
    priority: "medium",
    customer_name: "Beta Inc",
    site_location: "Office 2",
    device_ipn: "DSP-001",
    device_serial: "SN-456",
    reported_by: "Jane",
    reported_at: "2026-01-20",
    description: "Display flickers intermittently",
    root_cause: "",
    resolution: "",
    ncr_id: "",
    eco_id: "",
    created_at: "2026-01-20",
    updated_at: "2026-01-20",
  },
];

vi.mock("../lib/api", () => ({
  api: {
    getFieldReports: vi.fn().mockResolvedValue([]),
    createFieldReport: vi.fn(),
    updateFieldReport: vi.fn(),
    deleteFieldReport: vi.fn(),
    createNCRFromFieldReport: vi.fn(),
  },
}));

import { api } from "../lib/api";

describe("FieldReports", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (api.getFieldReports as ReturnType<typeof vi.fn>).mockResolvedValue(mockFieldReports);
  });

  it("renders the field reports page title", async () => {
    render(<FieldReports />);
    await waitFor(() => {
      expect(screen.getByText("Field Reports")).toBeInTheDocument();
    });
  });

  it("displays field reports in a table", async () => {
    render(<FieldReports />);
    await waitFor(() => {
      expect(screen.getByText("Motor overheating")).toBeInTheDocument();
    });
    expect(screen.getByText("Display flickering")).toBeInTheDocument();
    expect(screen.getByText("Acme Corp")).toBeInTheDocument();
  });

  it("shows status badges", async () => {
    render(<FieldReports />);
    await waitFor(() => {
      expect(screen.getByText("open")).toBeInTheDocument();
    });
    expect(screen.getByText("investigating")).toBeInTheDocument();
  });

  it("shows priority badges", async () => {
    render(<FieldReports />);
    await waitFor(() => {
      expect(screen.getByText("high")).toBeInTheDocument();
    });
    expect(screen.getByText("medium")).toBeInTheDocument();
  });

  it("shows create button", async () => {
    render(<FieldReports />);
    await waitFor(() => {
      expect(screen.getByText("Create Report")).toBeInTheDocument();
    });
  });

  it("opens create dialog when button clicked", async () => {
    render(<FieldReports />);
    await waitFor(() => {
      expect(screen.getByText("Create Report")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create Report"));
    await waitFor(() => {
      expect(screen.getByText("Create Field Report")).toBeInTheDocument();
    });
  });

  it("shows loading state initially", () => {
    (api.getFieldReports as ReturnType<typeof vi.fn>).mockReturnValue(new Promise(() => {}));
    render(<FieldReports />);
    expect(screen.getByText("Loading field reports...")).toBeInTheDocument();
  });

  it("shows empty state when no reports", async () => {
    (api.getFieldReports as ReturnType<typeof vi.fn>).mockResolvedValue([]);
    render(<FieldReports />);
    await waitFor(() => {
      expect(screen.getByText("No field reports found")).toBeInTheDocument();
    });
  });

  it("has filter selects for status and type", async () => {
    render(<FieldReports />);
    await waitFor(() => {
      expect(screen.getByText("Motor overheating")).toBeInTheDocument();
    });
    // Filter labels + table headers both have "Status" etc, use getAllByText
    expect(screen.getAllByText("Status").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("Type").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("Priority").length).toBeGreaterThanOrEqual(1);
  });
});
