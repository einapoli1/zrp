import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import userEvent from "@testing-library/user-event";
import { mockTestRecords } from "../test/mocks";

const mockGetTestRecords = vi.fn().mockResolvedValue(mockTestRecords);
const mockCreateTestRecord = vi.fn().mockResolvedValue(mockTestRecords[0]);

vi.mock("../lib/api", () => ({
  api: {
    getTestRecords: (...args: any[]) => mockGetTestRecords(...args),
    createTestRecord: (...args: any[]) => mockCreateTestRecord(...args),
  },
}));

import Testing from "./Testing";

beforeEach(() => vi.clearAllMocks());

describe("Testing", () => {
  it("renders loading state", () => {
    render(<Testing />);
    expect(screen.getByText("Loading test records...")).toBeInTheDocument();
  });

  it("renders page title and description", async () => {
    render(<Testing />);
    await waitFor(() => {
      expect(screen.getByText("Testing")).toBeInTheDocument();
      expect(screen.getByText("Track device testing results and quality metrics")).toBeInTheDocument();
    });
  });

  it("renders test records after loading", async () => {
    render(<Testing />);
    await waitFor(() => {
      expect(screen.getByText("SN-100")).toBeInTheDocument();
    });
    expect(screen.getByText("SN-101")).toBeInTheDocument();
  });

  it("shows test results with badges", async () => {
    render(<Testing />);
    await waitFor(() => {
      expect(screen.getByText("Pass")).toBeInTheDocument();
      expect(screen.getByText("Fail")).toBeInTheDocument();
    });
  });

  it("has Create Test Record button", async () => {
    render(<Testing />);
    await waitFor(() => {
      expect(screen.getByText("Create Test Record")).toBeInTheDocument();
    });
  });

  it("shows empty state", async () => {
    mockGetTestRecords.mockResolvedValueOnce([]);
    render(<Testing />);
    await waitFor(() => {
      expect(screen.getByText(/no test records found/i)).toBeInTheDocument();
    });
  });

  it("shows Test Records card title", async () => {
    render(<Testing />);
    await waitFor(() => {
      expect(screen.getByText("Test Records")).toBeInTheDocument();
    });
  });

  it("shows table headers", async () => {
    render(<Testing />);
    await waitFor(() => {
      expect(screen.getByText("Test ID")).toBeInTheDocument();
      expect(screen.getByText("Device S/N")).toBeInTheDocument();
      expect(screen.getByText("IPN")).toBeInTheDocument();
      expect(screen.getByText("Test Type")).toBeInTheDocument();
      expect(screen.getByText("Result")).toBeInTheDocument();
      expect(screen.getByText("Tested By")).toBeInTheDocument();
      expect(screen.getByText("Date")).toBeInTheDocument();
    });
  });

  it("shows statistics cards", async () => {
    render(<Testing />);
    await waitFor(() => {
      expect(screen.getByText("Total Tests")).toBeInTheDocument();
      expect(screen.getByText("Passed")).toBeInTheDocument();
      expect(screen.getByText("Failed")).toBeInTheDocument();
      expect(screen.getByText("Pass Rate")).toBeInTheDocument();
    });
  });

  it("shows correct statistics counts", async () => {
    render(<Testing />);
    await waitFor(() => {
      expect(screen.getByText("2")).toBeInTheDocument(); // total
      expect(screen.getByText("50%")).toBeInTheDocument(); // pass rate (1/2)
    });
  });

  it("shows test type values", async () => {
    render(<Testing />);
    await waitFor(() => {
      expect(screen.getByText("functional")).toBeInTheDocument();
      expect(screen.getByText("burn-in")).toBeInTheDocument();
    });
  });

  it("shows tested_by values", async () => {
    render(<Testing />);
    await waitFor(() => {
      expect(screen.getByText("tech1")).toBeInTheDocument();
      expect(screen.getByText("tech2")).toBeInTheDocument();
    });
  });

  it("shows test IDs formatted as TEST-XXXX", async () => {
    render(<Testing />);
    await waitFor(() => {
      expect(screen.getByText("TEST-0001")).toBeInTheDocument();
      expect(screen.getByText("TEST-0002")).toBeInTheDocument();
    });
  });

  it("opens create dialog when button clicked", async () => {
    const user = userEvent.setup();
    render(<Testing />);
    await waitFor(() => {
      expect(screen.getByText("Create Test Record")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Create Test Record"));
    await waitFor(() => {
      expect(screen.getByPlaceholderText("Serial number")).toBeInTheDocument();
      expect(screen.getByPlaceholderText("Internal part number")).toBeInTheDocument();
    });
  });

  it("shows form fields in create dialog", async () => {
    const user = userEvent.setup();
    render(<Testing />);
    await waitFor(() => {
      expect(screen.getByText("Create Test Record")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Create Test Record"));
    await waitFor(() => {
      expect(screen.getByPlaceholderText("Serial number")).toBeInTheDocument();
      expect(screen.getByPlaceholderText("Internal part number")).toBeInTheDocument();
      expect(screen.getByPlaceholderText("e.g., v1.2.3")).toBeInTheDocument();
      expect(screen.getByPlaceholderText("Technician name")).toBeInTheDocument();
    });
  });

  it("shows IPN values in table", async () => {
    render(<Testing />);
    await waitFor(() => {
      const ipnCells = screen.getAllByText("IPN-003");
      expect(ipnCells.length).toBeGreaterThan(0);
    });
  });

  it("calls getTestRecords on mount", async () => {
    render(<Testing />);
    await waitFor(() => {
      expect(mockGetTestRecords).toHaveBeenCalledTimes(1);
    });
  });

  it("fills create test record form end-to-end and submits", async () => {
    const user = userEvent.setup();
    render(<Testing />);
    await waitFor(() => expect(screen.getByText("Create Test Record")).toBeInTheDocument());

    await user.click(screen.getByText("Create Test Record"));
    await waitFor(() => expect(screen.getByPlaceholderText("Serial number")).toBeInTheDocument());

    await user.type(screen.getByPlaceholderText("Serial number"), "SN-999");
    await user.type(screen.getByPlaceholderText("Internal part number"), "IPN-010");
    await user.type(screen.getByPlaceholderText("e.g., v1.2.3"), "2.0.0");
    await user.type(screen.getByPlaceholderText("Technician name"), "tech_new");
    await user.type(screen.getByPlaceholderText("Test measurements and data"), "All good");
    await user.type(screen.getByPlaceholderText("Additional notes or observations"), "No issues");

    // Submit the form
    const dialog = screen.getByRole("dialog");
    const submitBtn = Array.from(dialog.querySelectorAll("button")).find(b => b.textContent === "Create Test Record");
    await user.click(submitBtn!);

    await waitFor(() => {
      expect(mockCreateTestRecord).toHaveBeenCalledTimes(1);
      expect(mockCreateTestRecord).toHaveBeenCalledWith(
        expect.objectContaining({
          serial_number: "SN-999",
          ipn: "IPN-010",
          firmware_version: "2.0.0",
          tested_by: "tech_new",
        })
      );
    });
  });

  it("validates required fields — form uses native required validation", async () => {
    const user = userEvent.setup();
    render(<Testing />);
    await waitFor(() => expect(screen.getByText("Create Test Record")).toBeInTheDocument());

    await user.click(screen.getByText("Create Test Record"));
    await waitFor(() => expect(screen.getByPlaceholderText("Serial number")).toBeInTheDocument());

    // The serial_number and ipn inputs have `required` attribute
    const serialInput = screen.getByPlaceholderText("Serial number");
    expect(serialInput).toBeRequired();
    const ipnInput = screen.getByPlaceholderText("Internal part number");
    expect(ipnInput).toBeRequired();
  });

  it("handles getTestRecords API rejection gracefully", async () => {
    mockGetTestRecords.mockRejectedValueOnce(new Error("Network error"));
    render(<Testing />);
    await waitFor(() => {
      expect(screen.queryByText("Loading test records...")).not.toBeInTheDocument();
    });
    expect(screen.getByText(/no test records found/i)).toBeInTheDocument();
  });

  it("handles createTestRecord API rejection gracefully", async () => {
    const user = userEvent.setup();
    mockCreateTestRecord.mockRejectedValueOnce(new Error("Create failed"));
    render(<Testing />);
    await waitFor(() => {
      expect(screen.getByText("SN-100")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Create Test Record"));
    await waitFor(() => {
      expect(screen.getByPlaceholderText("Serial number")).toBeInTheDocument();
    });
    await user.type(screen.getByPlaceholderText("Serial number"), "SN-999");
    await user.type(screen.getByPlaceholderText("Internal part number"), "IPN-X");
    const dialog = screen.getByRole("dialog");
    const submitBtn = Array.from(dialog.querySelectorAll("button")).find(b => b.textContent === "Create Test Record");
    await user.click(submitBtn!);
    await waitFor(() => {
      expect(mockCreateTestRecord).toHaveBeenCalled();
    });
    // Should not crash — original records still visible
    expect(screen.getByText("SN-100")).toBeInTheDocument();
  });
});
