import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";

const mockGetRFQs = vi.fn();
const mockCreateRFQ = vi.fn();

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useNavigate: () => mockNavigate };
});

vi.mock("../lib/api", () => ({
  api: {
    getRFQs: (...args: any[]) => mockGetRFQs(...args),
    createRFQ: (...args: any[]) => mockCreateRFQ(...args),
  },
}));

import RFQs from "./RFQs";

const mockRFQsList = [
  { id: "RFQ-2026-0001", title: "Resistor Bulk Quote", status: "draft", created_by: "admin", created_at: "2026-01-15", updated_at: "2026-01-15", due_date: "2026-02-01", notes: "" },
  { id: "RFQ-2026-0002", title: "MCU Sourcing", status: "sent", created_by: "admin", created_at: "2026-01-20", updated_at: "2026-01-21", due_date: "2026-02-15", notes: "" },
];

beforeEach(() => {
  vi.clearAllMocks();
  mockGetRFQs.mockResolvedValue(mockRFQsList);
  mockCreateRFQ.mockResolvedValue(mockRFQsList[0]);
});

describe("RFQs", () => {
  it("renders loading state", () => {
    render(<RFQs />);
    expect(screen.getByText(/loading rfqs/i)).toBeInTheDocument();
  });

  it("renders RFQ list after loading", async () => {
    render(<RFQs />);
    await waitFor(() => {
      expect(screen.getByText("Resistor Bulk Quote")).toBeInTheDocument();
    });
    expect(screen.getByText("MCU Sourcing")).toBeInTheDocument();
  });

  it("shows empty state when no RFQs", async () => {
    mockGetRFQs.mockResolvedValueOnce([]);
    render(<RFQs />);
    await waitFor(() => {
      expect(screen.getByText(/no rfqs found/i)).toBeInTheDocument();
    });
  });

  it("has Create RFQ button", async () => {
    render(<RFQs />);
    await waitFor(() => {
      expect(screen.getByText("Create RFQ")).toBeInTheDocument();
    });
  });

  it("opens create dialog on button click", async () => {
    render(<RFQs />);
    await waitFor(() => screen.getByText("Create RFQ"));
    fireEvent.click(screen.getByText("Create RFQ"));
    await waitFor(() => {
      expect(screen.getByText("New RFQ")).toBeInTheDocument();
    });
  });

  it("creates RFQ and navigates to detail", async () => {
    render(<RFQs />);
    await waitFor(() => screen.getByText("Create RFQ"));
    fireEvent.click(screen.getByText("Create RFQ"));
    await waitFor(() => screen.getByLabelText("Title"));
    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "New RFQ" } });
    fireEvent.click(screen.getByRole("button", { name: "Create" }));
    await waitFor(() => {
      expect(mockCreateRFQ).toHaveBeenCalledWith(expect.objectContaining({ title: "New RFQ" }));
      expect(mockNavigate).toHaveBeenCalledWith(`/rfqs/${mockRFQsList[0].id}`);
    });
  });

  it("shows status badges", async () => {
    render(<RFQs />);
    await waitFor(() => {
      expect(screen.getByText("draft")).toBeInTheDocument();
      expect(screen.getByText("sent")).toBeInTheDocument();
    });
  });
});
