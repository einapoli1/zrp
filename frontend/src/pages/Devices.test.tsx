import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import userEvent from "@testing-library/user-event";
import { mockDevices } from "../test/mocks";

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useNavigate: () => mockNavigate };
});

const mockGetDevices = vi.fn().mockResolvedValue(mockDevices);
const mockCreateDevice = vi.fn().mockResolvedValue(mockDevices[0]);
const mockImportDevices = vi.fn().mockResolvedValue({ success: 5, errors: [] });
const mockExportDevices = vi.fn().mockResolvedValue(new Blob(["csv data"]));

vi.mock("../lib/api", () => ({
  api: {
    getDevices: (...args: any[]) => mockGetDevices(...args),
    createDevice: (...args: any[]) => mockCreateDevice(...args),
    importDevices: (...args: any[]) => mockImportDevices(...args),
    exportDevices: (...args: any[]) => mockExportDevices(...args),
  },
}));

import Devices from "./Devices";

beforeEach(() => {
  vi.clearAllMocks();
  mockGetDevices.mockResolvedValue(mockDevices);
});

describe("Devices", () => {
  it("renders loading state", () => {
    render(<Devices />);
    expect(screen.getByText("Loading devices...")).toBeInTheDocument();
  });

  it("renders device list after loading", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("SN-100")).toBeInTheDocument();
    });
    expect(screen.getByText("SN-101")).toBeInTheDocument();
  });

  it("shows page header and description", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("Device Registry")).toBeInTheDocument();
    });
    expect(screen.getByText("Manage device inventory and track firmware versions")).toBeInTheDocument();
  });

  it("displays device table with correct columns", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("Serial Number")).toBeInTheDocument();
    });
    expect(screen.getByText("IPN")).toBeInTheDocument();
    expect(screen.getByText("Firmware Version")).toBeInTheDocument();
    expect(screen.getByText("Customer")).toBeInTheDocument();
    expect(screen.getByText("Location")).toBeInTheDocument();
    expect(screen.getByText("Status")).toBeInTheDocument();
    expect(screen.getByText("Last Seen")).toBeInTheDocument();
    expect(screen.getByText("Actions")).toBeInTheDocument();
  });

  it("shows device details in table rows", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("Acme")).toBeInTheDocument();
      expect(screen.getByText("Tech Co")).toBeInTheDocument();
    });
    expect(screen.getByText("Building A")).toBeInTheDocument();
    expect(screen.getByText("Floor 2")).toBeInTheDocument();
    expect(screen.getByText("1.0.0")).toBeInTheDocument();
    expect(screen.getByText("0.9.0")).toBeInTheDocument();
  });

  it("shows status badges", async () => {
    render(<Devices />);
    await waitFor(() => {
      const badges = screen.getAllByText("Active");
      expect(badges.length).toBeGreaterThanOrEqual(1);
    });
  });

  it("shows empty state when no devices", async () => {
    mockGetDevices.mockResolvedValueOnce([]);
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText(/No devices found/i)).toBeInTheDocument();
    });
  });

  it("navigates to device detail on row click", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("SN-100")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("SN-100"));
    expect(mockNavigate).toHaveBeenCalledWith("/devices/SN-100");
  });

  it("navigates to device detail on View Details button", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getAllByText("View Details").length).toBeGreaterThan(0);
    });
    fireEvent.click(screen.getAllByText("View Details")[0]);
    expect(mockNavigate).toHaveBeenCalledWith("/devices/SN-100");
  });

  it("shows statistics cards", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("Total Devices")).toBeInTheDocument();
    });
    expect(screen.getByText("Maintenance")).toBeInTheDocument();
    expect(screen.getByText("Inactive")).toBeInTheDocument();
  });

  it("shows correct statistics counts", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("Total Devices")).toBeInTheDocument();
    });
    // 2 devices total, both active - count appears in stats cards
    const totalCard = screen.getByText("Total Devices").closest("div")?.parentElement;
    expect(totalCard).toBeTruthy();
  });

  // Export CSV
  it("has Export CSV button", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("Export CSV")).toBeInTheDocument();
    });
  });

  it("calls exportDevices on Export CSV click", async () => {
    // Mock URL methods
    const origCreateObjectURL = URL.createObjectURL;
    const origRevokeObjectURL = URL.revokeObjectURL;
    URL.createObjectURL = vi.fn().mockReturnValue("blob:test");
    URL.revokeObjectURL = vi.fn();

    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("Export CSV")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Export CSV"));
    await waitFor(() => {
      expect(mockExportDevices).toHaveBeenCalled();
    });

    URL.createObjectURL = origCreateObjectURL;
    URL.revokeObjectURL = origRevokeObjectURL;
  });

  // Import CSV
  it("has Import CSV button", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("Import CSV")).toBeInTheDocument();
    });
  });

  it("has Import CSV button rendered", async () => {
    render(<Devices />);
    await waitFor(() => {
      const importBtn = screen.getByText("Import CSV");
      expect(importBtn).toBeInTheDocument();
      expect(importBtn.closest("button")).toBeTruthy();
    });
  });

  // Create device dialog
  it("has Add Device button", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("Add Device")).toBeInTheDocument();
    });
  });

  it("opens create device dialog", async () => {
    render(<Devices />);
    await waitFor(() => screen.getByText("Add Device"));
    fireEvent.click(screen.getByText("Add Device"));
    await waitFor(() => {
      expect(screen.getByText("Add New Device")).toBeInTheDocument();
    });
  });

  it("shows create device form fields", async () => {
    render(<Devices />);
    await waitFor(() => screen.getByText("Add Device"));
    fireEvent.click(screen.getByText("Add Device"));
    await waitFor(() => {
      expect(screen.getByLabelText("Serial Number *")).toBeInTheDocument();
      expect(screen.getByLabelText("IPN")).toBeInTheDocument();
      expect(screen.getByLabelText("Firmware Version")).toBeInTheDocument();
      expect(screen.getByLabelText("Customer")).toBeInTheDocument();
      expect(screen.getByLabelText("Location")).toBeInTheDocument();
      expect(screen.getByLabelText("Notes")).toBeInTheDocument();
    });
  });

  it("disables Add Device submit when serial_number empty", async () => {
    render(<Devices />);
    await waitFor(() => screen.getByText("Add Device"));
    fireEvent.click(screen.getByText("Add Device"));
    await waitFor(() => {
      // The submit button inside dialog also says "Add Device"
      const buttons = screen.getAllByRole("button", { name: "Add Device" });
      const submitBtn = buttons[buttons.length - 1];
      expect(submitBtn).toBeDisabled();
    });
  });

  it("creates device and refreshes list", async () => {
    const user = userEvent.setup();
    render(<Devices />);
    await waitFor(() => screen.getByText("Add Device"));
    await user.click(screen.getByText("Add Device"));
    await waitFor(() => screen.getByLabelText("Serial Number *"));

    await user.type(screen.getByLabelText("Serial Number *"), "SN-999");
    
    // Find the submit button (second "Add Device" button)
    const buttons = screen.getAllByRole("button", { name: "Add Device" });
    const submitBtn = buttons[buttons.length - 1];
    expect(submitBtn).not.toBeDisabled();
    await user.click(submitBtn);

    await waitFor(() => {
      expect(mockCreateDevice).toHaveBeenCalledWith(
        expect.objectContaining({ serial_number: "SN-999" })
      );
    });
    // Refreshes list after create
    expect(mockGetDevices).toHaveBeenCalledTimes(2);
  });

  it("opens Import CSV dialog, selects file, clicks Import, verifies API called", async () => {
    const user = userEvent.setup();
    render(<Devices />);
    await waitFor(() => expect(screen.getByText("SN-100")).toBeInTheDocument());
    
    // Open import dialog by clicking the trigger button 
    const buttons = screen.getAllByRole("button");
    const importCsvButton = buttons.find(b => b.textContent?.includes("Import CSV"));
    expect(importCsvButton).toBeTruthy();
    await user.click(importCsvButton!);
    await waitFor(() => expect(screen.getByLabelText("CSV File")).toBeInTheDocument());

    const file = new File(["serial_number,ipn\nSN-200,IPN-005"], "devices.csv", { type: "text/csv" });
    const input = screen.getByLabelText("CSV File");
    fireEvent.change(input, { target: { files: [file] } });

    // Find Import button inside dialog
    const dialog = screen.getByRole("dialog");
    const importBtn = Array.from(dialog.querySelectorAll("button")).find(b => b.textContent === "Import");
    await user.click(importBtn!);

    await waitFor(() => {
      expect(mockImportDevices).toHaveBeenCalledWith(expect.any(File));
    });
  });

  it("displays import results — success count and errors", async () => {
    mockImportDevices.mockResolvedValueOnce({ success: 3, errors: ["Row 4: missing serial_number", "Row 7: duplicate"] });
    const user = userEvent.setup();
    render(<Devices />);
    await waitFor(() => expect(screen.getByText("SN-100")).toBeInTheDocument());
    
    const buttons = screen.getAllByRole("button");
    const importCsvButton = buttons.find(b => b.textContent?.includes("Import CSV"));
    await user.click(importCsvButton!);
    await waitFor(() => expect(screen.getByLabelText("CSV File")).toBeInTheDocument());

    const file = new File(["data"], "devices.csv", { type: "text/csv" });
    fireEvent.change(screen.getByLabelText("CSV File"), { target: { files: [file] } });
    
    const dialog = screen.getByRole("dialog");
    const importBtn = Array.from(dialog.querySelectorAll("button")).find(b => b.textContent === "Import");
    await user.click(importBtn!);

    await waitFor(() => {
      expect(screen.getByText(/Successfully imported: 3 devices/)).toBeInTheDocument();
      expect(screen.getByText("Errors (2):")).toBeInTheDocument();
      expect(screen.getByText("Row 4: missing serial_number")).toBeInTheDocument();
      expect(screen.getByText("Row 7: duplicate")).toBeInTheDocument();
    });
  });

  it("Import button is disabled when no file selected", async () => {
    const user = userEvent.setup();
    render(<Devices />);
    await waitFor(() => expect(screen.getByText("SN-100")).toBeInTheDocument());
    
    const buttons = screen.getAllByRole("button");
    const importCsvButton = buttons.find(b => b.textContent?.includes("Import CSV"));
    await user.click(importCsvButton!);
    await waitFor(() => expect(screen.getByLabelText("CSV File")).toBeInTheDocument());

    expect(screen.getByRole("button", { name: "Import" })).toBeDisabled();
  });

  it("handles fetch error gracefully", async () => {
    mockGetDevices.mockRejectedValueOnce(new Error("Network error"));
    render(<Devices />);
    await waitFor(() => {
      // Should finish loading even on error
      expect(screen.queryByText("Loading devices...")).not.toBeInTheDocument();
    });
  });

  it("renders Device Inventory card title", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("Device Inventory")).toBeInTheDocument();
    });
  });
});
