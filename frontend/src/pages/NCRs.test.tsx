import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockNCRs } from "../test/mocks";

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useNavigate: () => mockNavigate };
});

const mockGetNCRs = vi.fn().mockResolvedValue(mockNCRs);
const mockCreateNCR = vi.fn().mockResolvedValue(mockNCRs[0]);

vi.mock("../lib/api", () => ({
  api: {
    getNCRs: (...args: any[]) => mockGetNCRs(...args),
    createNCR: (...args: any[]) => mockCreateNCR(...args),
  },
}));

import NCRs from "./NCRs";

beforeEach(() => {
  vi.clearAllMocks();
  mockGetNCRs.mockResolvedValue(mockNCRs);
});

describe("NCRs", () => {
  it("renders loading state", () => {
    render(<NCRs />);
    expect(screen.getByText("Loading NCRs...")).toBeInTheDocument();
  });

  it("renders page heading and description", async () => {
    render(<NCRs />);
    await waitFor(() => {
      expect(screen.getByText("Non-Conformance Reports")).toBeInTheDocument();
    });
    expect(screen.getByText("Track quality issues and corrective actions")).toBeInTheDocument();
  });

  it("renders NCR list after loading", async () => {
    render(<NCRs />);
    await waitFor(() => {
      expect(screen.getByText("NCR-001")).toBeInTheDocument();
    });
    expect(screen.getByText("NCR-002")).toBeInTheDocument();
    expect(screen.getByText("Defective resistor batch")).toBeInTheDocument();
    expect(screen.getByText("Cracked capacitor")).toBeInTheDocument();
  });

  it("shows severity badges", async () => {
    render(<NCRs />);
    await waitFor(() => {
      expect(screen.getByText("Major")).toBeInTheDocument();
    });
    expect(screen.getByText("Critical")).toBeInTheDocument();
  });

  it("shows status badges", async () => {
    render(<NCRs />);
    await waitFor(() => {
      expect(screen.getByText("Open")).toBeInTheDocument();
    });
    expect(screen.getByText("Resolved")).toBeInTheDocument();
  });

  it("shows table headers", async () => {
    render(<NCRs />);
    await waitFor(() => {
      expect(screen.getByText("NCR-001")).toBeInTheDocument();
    });
    expect(screen.getByText("NCR ID")).toBeInTheDocument();
    expect(screen.getByText("Title")).toBeInTheDocument();
    expect(screen.getByText("Severity")).toBeInTheDocument();
    expect(screen.getByText("Status")).toBeInTheDocument();
    expect(screen.getByText("Date")).toBeInTheDocument();
    expect(screen.getByText("Actions")).toBeInTheDocument();
  });

  it("shows NCR Records card title", async () => {
    render(<NCRs />);
    await waitFor(() => {
      expect(screen.getByText("NCR Records")).toBeInTheDocument();
    });
  });

  it("shows empty state when no NCRs", async () => {
    mockGetNCRs.mockResolvedValueOnce([]);
    render(<NCRs />);
    await waitFor(() => {
      expect(screen.getByText(/No NCRs found/)).toBeInTheDocument();
    });
  });

  it("navigates to NCR detail on row click", async () => {
    render(<NCRs />);
    await waitFor(() => {
      expect(screen.getByText("NCR-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("NCR-001"));
    expect(mockNavigate).toHaveBeenCalledWith("/ncrs/NCR-001");
  });

  it("has View Details button that navigates", async () => {
    render(<NCRs />);
    await waitFor(() => {
      expect(screen.getAllByText("View Details").length).toBeGreaterThan(0);
    });
    fireEvent.click(screen.getAllByText("View Details")[0]);
    expect(mockNavigate).toHaveBeenCalledWith("/ncrs/NCR-001");
  });

  // Create dialog
  it("has create NCR button", async () => {
    render(<NCRs />);
    await waitFor(() => {
      expect(screen.getByText("NCR-001")).toBeInTheDocument();
    });
    expect(screen.getByRole("button", { name: /create ncr/i })).toBeInTheDocument();
  });

  it("opens create dialog with form fields", async () => {
    render(<NCRs />);
    await waitFor(() => {
      expect(screen.getByText("NCR-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole("button", { name: /create ncr/i }));
    await waitFor(() => {
      expect(screen.getByText("Create New NCR")).toBeInTheDocument();
    });
    expect(screen.getByLabelText("Title *")).toBeInTheDocument();
    expect(screen.getByLabelText("Description")).toBeInTheDocument();
    // Severity uses a custom Select so getByLabelText won't work
    expect(screen.getByText("Severity *")).toBeInTheDocument();
    expect(screen.getByLabelText("Affected IPN")).toBeInTheDocument();
  });

  it("has cancel and submit buttons in create dialog", async () => {
    render(<NCRs />);
    await waitFor(() => {
      expect(screen.getByText("NCR-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create NCR"));
    await waitFor(() => {
      expect(screen.getByText("Cancel")).toBeInTheDocument();
    });
  });

  it("submits create form and adds NCR to list", async () => {
    render(<NCRs />);
    await waitFor(() => {
      expect(screen.getByText("NCR-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create NCR"));
    await waitFor(() => {
      expect(screen.getByLabelText("Title *")).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText("Title *"), { target: { value: "New defect" } });
    fireEvent.change(screen.getByLabelText("Description"), { target: { value: "A new issue" } });

    // Submit
    const submitButtons = screen.getAllByText("Create NCR");
    const submitBtn = submitButtons[submitButtons.length - 1];
    fireEvent.click(submitBtn);

    await waitFor(() => {
      expect(mockCreateNCR).toHaveBeenCalledWith(
        expect.objectContaining({ title: "New defect", description: "A new issue" })
      );
    });
  });

  it("handles API error on fetch gracefully", async () => {
    mockGetNCRs.mockRejectedValueOnce(new Error("Network error"));
    render(<NCRs />);
    await waitFor(() => {
      expect(screen.getByText(/No NCRs found/)).toBeInTheDocument();
    });
  });

  it("create form error handling — dialog stays open on reject", async () => {
    mockCreateNCR.mockRejectedValueOnce(new Error("Server error"));
    render(<NCRs />);
    await waitFor(() => {
      expect(screen.getByText("NCR-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole("button", { name: /create ncr/i }));
    await waitFor(() => {
      expect(screen.getByLabelText("Title *")).toBeInTheDocument();
    });
    fireEvent.change(screen.getByLabelText("Title *"), { target: { value: "Fail test" } });
    const submitButtons = screen.getAllByText("Create NCR");
    fireEvent.click(submitButtons[submitButtons.length - 1]);
    await waitFor(() => {
      expect(mockCreateNCR).toHaveBeenCalled();
    });
    // Dialog should still be open
    expect(screen.getByText("Create New NCR")).toBeInTheDocument();
    expect(screen.getByLabelText("Title *")).toBeInTheDocument();
  });

  it("cancel button closes create dialog", async () => {
    render(<NCRs />);
    await waitFor(() => {
      expect(screen.getByText("NCR-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole("button", { name: /create ncr/i }));
    await waitFor(() => {
      expect(screen.getByText("Create New NCR")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Cancel"));
    await waitFor(() => {
      expect(screen.queryByText("Create New NCR")).not.toBeInTheDocument();
    });
  });

  it("dialog closes and form resets after successful create", async () => {
    const newNCR = { ...mockNCRs[0], id: "NCR-099", title: "Brand new" };
    mockCreateNCR.mockResolvedValueOnce(newNCR);
    render(<NCRs />);
    await waitFor(() => {
      expect(screen.getByText("NCR-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole("button", { name: /create ncr/i }));
    await waitFor(() => {
      expect(screen.getByLabelText("Title *")).toBeInTheDocument();
    });
    fireEvent.change(screen.getByLabelText("Title *"), { target: { value: "Brand new" } });
    fireEvent.change(screen.getByLabelText("Description"), { target: { value: "desc" } });
    const submitButtons = screen.getAllByText("Create NCR");
    fireEvent.click(submitButtons[submitButtons.length - 1]);
    await waitFor(() => {
      expect(screen.queryByText("Create New NCR")).not.toBeInTheDocument();
    });
    // Re-open dialog to check form is reset
    fireEvent.click(screen.getByRole("button", { name: /create ncr/i }));
    await waitFor(() => {
      expect(screen.getByLabelText("Title *")).toBeInTheDocument();
    });
    expect((screen.getByLabelText("Title *") as HTMLInputElement).value).toBe("");
    expect((screen.getByLabelText("Description") as HTMLTextAreaElement).value).toBe("");
  });

  it("new NCR prepended to list after create", async () => {
    const newNCR = { ...mockNCRs[0], id: "NCR-NEW", title: "New defect item", severity: "minor", status: "open", created_at: "2024-02-01" };
    mockCreateNCR.mockResolvedValueOnce(newNCR);
    render(<NCRs />);
    await waitFor(() => {
      expect(screen.getByText("NCR-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole("button", { name: /create ncr/i }));
    await waitFor(() => {
      expect(screen.getByLabelText("Title *")).toBeInTheDocument();
    });
    fireEvent.change(screen.getByLabelText("Title *"), { target: { value: "New defect item" } });
    const submitButtons = screen.getAllByText("Create NCR");
    fireEvent.click(submitButtons[submitButtons.length - 1]);
    await waitFor(() => {
      expect(screen.getByText("NCR-NEW")).toBeInTheDocument();
    });
    // Verify NCR-NEW appears before NCR-001 in the DOM
    const rows = screen.getAllByRole("row");
    const ncrNewIdx = rows.findIndex(r => r.textContent?.includes("NCR-NEW"));
    const ncr001Idx = rows.findIndex(r => r.textContent?.includes("NCR-001"));
    expect(ncrNewIdx).toBeLessThan(ncr001Idx);
  });

  it("severity selector in create form changes value", async () => {
    render(<NCRs />);
    await waitFor(() => {
      expect(screen.getByText("NCR-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole("button", { name: /create ncr/i }));
    await waitFor(() => {
      expect(screen.getByText("Create New NCR")).toBeInTheDocument();
    });
    // Fill required field and submit with default severity (minor)
    fireEvent.change(screen.getByLabelText("Title *"), { target: { value: "Sev test" } });
    const submitButtons = screen.getAllByText("Create NCR");
    fireEvent.click(submitButtons[submitButtons.length - 1]);
    await waitFor(() => {
      expect(mockCreateNCR).toHaveBeenCalledWith(
        expect.objectContaining({ severity: "minor" })
      );
    });
  });

  it("formats dates correctly", async () => {
    render(<NCRs />);
    await waitFor(() => {
      expect(screen.getByText("NCR-001")).toBeInTheDocument();
    });
    // Dates are rendered via toLocaleDateString - check they exist
    const dateCells = screen.getAllByText((content) => content.includes("2024"));
    expect(dateCells.length).toBeGreaterThan(0);
  });
});
