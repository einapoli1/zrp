import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "../test/test-utils";
import PartDetail from "./PartDetail";
import { api, type Part, type PartManufacturer, type Manufacturer } from "../lib/api";
import { toast } from "sonner";
import { useParams } from "react-router-dom";

vi.mock("../lib/api");
vi.mock("sonner");
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: vi.fn(),
    useNavigate: vi.fn(() => vi.fn()),
  };
});

const mockPart: Part = {
  ipn: "IC-001",
  category: "ICs",
  description: "Test IC",
  fields: {
    _category: "ICs",
    description: "Test IC",
  },
};

const mockAvailableManufacturers: Manufacturer[] = [
  {
    id: 1,
    name: "Texas Instruments",
    contact_email: "sales@ti.com",
    approved: true,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
  },
  {
    id: 2,
    name: "Analog Devices",
    contact_email: "info@analog.com",
    approved: true,
    created_at: "2024-01-02T00:00:00Z",
    updated_at: "2024-01-02T00:00:00Z",
  },
];

const mockPartManufacturers: PartManufacturer[] = [
  {
    id: 1,
    part_id: 1,
    manufacturer_id: 1,
    manufacturer_name: "Texas Instruments",
    mpn: "TPS54560ADDAR",
    is_primary: true,
    approved: true,
    notes: "Primary source",
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
  },
  {
    id: 2,
    part_id: 1,
    manufacturer_id: 2,
    manufacturer_name: "Analog Devices",
    mpn: "LT8610EMSE",
    is_primary: false,
    approved: true,
    notes: "Secondary source",
    created_at: "2024-01-02T00:00:00Z",
    updated_at: "2024-01-02T00:00:00Z",
  },
];

describe("PartDetail - Manufacturers Section", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(useParams).mockReturnValue({ ipn: "IC-001" });
    vi.mocked(api.getPart).mockResolvedValue(mockPart);
    vi.mocked(api.getPartManufacturers).mockResolvedValue({
      manufacturers: mockPartManufacturers,
      count: mockPartManufacturers.length,
    });
    vi.mocked(api.getManufacturers).mockResolvedValue(mockAvailableManufacturers);
    vi.mocked(api.getPartCost).mockResolvedValue({} as any);
    vi.mocked(api.getPartWhereUsed).mockResolvedValue([]);
    vi.mocked(api.getPartChanges).mockResolvedValue([]);
    vi.mocked(api.getGitPLMConfig).mockResolvedValue({ base_url: "" });
  });

  describe("Render manufacturers list", () => {
    it("should display manufacturers table with normalized data", async () => {
      render(<PartDetail />);

      await waitFor(() => {
        expect(screen.getByText("Texas Instruments")).toBeInTheDocument();
        expect(screen.getByText("Analog Devices")).toBeInTheDocument();
        expect(screen.getByText("TPS54560ADDAR")).toBeInTheDocument();
        expect(screen.getByText("LT8610EMSE")).toBeInTheDocument();
      });

      // Check primary badge
      expect(screen.getByTestId("primary-badge-1")).toBeInTheDocument();
      expect(screen.getByText("Primary")).toBeInTheDocument();

      // Check approved checkmarks
      expect(screen.getByTestId("approved-check-1")).toBeInTheDocument();
      expect(screen.getByTestId("approved-check-2")).toBeInTheDocument();
    });

    it("should show empty state when no manufacturers exist", async () => {
      vi.mocked(api.getPartManufacturers).mockResolvedValue({
        manufacturers: [],
        count: 0,
      });

      render(<PartDetail />);

      await waitFor(() => {
        expect(screen.getByText("No manufacturers added")).toBeInTheDocument();
      });
    });
  });

  describe("Add manufacturer with dropdown", () => {
    it("should show autocomplete dropdown when adding manufacturer", async () => {
      render(<PartDetail />);

      await waitFor(() => {
        expect(screen.getByText("Texas Instruments")).toBeInTheDocument();
      });

      // Open add dialog
      fireEvent.click(screen.getByTestId("add-manufacturer-btn"));

      await waitFor(() => {
        expect(screen.getByTestId("manufacturer-input")).toBeInTheDocument();
      });

      // Type to trigger dropdown
      const input = screen.getByTestId("manufacturer-input");
      fireEvent.change(input, { target: { value: "Texas" } });

      await waitFor(() => {
        expect(screen.getByTestId("manufacturer-dropdown")).toBeInTheDocument();
        expect(screen.getByTestId("manufacturer-option-1")).toBeInTheDocument();
        expect(screen.getByText("sales@ti.com")).toBeInTheDocument();
      });
    });

    it("should add manufacturer via dropdown selection", async () => {
      vi.mocked(api.createPartManufacturer).mockResolvedValue({
        id: 3,
        message: "Manufacturer added successfully",
      });

      render(<PartDetail />);

      await waitFor(() => {
        expect(screen.getByText("Texas Instruments")).toBeInTheDocument();
      });

      // Open add dialog
      fireEvent.click(screen.getByTestId("add-manufacturer-btn"));

      await waitFor(() => {
        expect(screen.getByTestId("manufacturer-input")).toBeInTheDocument();
      });

      // Focus input to show dropdown
      const input = screen.getByTestId("manufacturer-input");
      fireEvent.focus(input);

      await waitFor(() => {
        expect(screen.getByTestId("manufacturer-dropdown")).toBeInTheDocument();
      });

      // Select manufacturer
      fireEvent.click(screen.getByTestId("manufacturer-option-1"));

      // Fill MPN
      fireEvent.change(screen.getByTestId("mpn-input"), {
        target: { value: "TPS54560ADDAR" },
      });

      // Save
      fireEvent.click(screen.getByTestId("save-manufacturer-btn"));

      await waitFor(() => {
        expect(api.createPartManufacturer).toHaveBeenCalledWith("IC-001", {
          manufacturer_id: 1,
          mpn: "TPS54560ADDAR",
          is_primary: false,
          approved: true,
          notes: "",
        });
        expect(toast.success).toHaveBeenCalledWith("Manufacturer added successfully");
      });
    });
  });

  describe("Edit manufacturer", () => {
    it("should edit existing manufacturer", async () => {
      vi.mocked(api.updatePartManufacturer).mockResolvedValue({
        message: "Manufacturer updated successfully",
      });

      render(<PartDetail />);

      await waitFor(() => {
        expect(screen.getByText("Texas Instruments")).toBeInTheDocument();
      });

      // Open edit dialog
      fireEvent.click(screen.getByTestId("edit-manufacturer-1"));

      await waitFor(() => {
        expect(screen.getByTestId("manufacturer-input")).toBeInTheDocument();
        expect(screen.getByTestId("mpn-input")).toHaveValue("TPS54560ADDAR");
      });

      // Update MPN
      fireEvent.change(screen.getByTestId("mpn-input"), {
        target: { value: "TPS54560BDDAR" },
      });

      // Save
      fireEvent.click(screen.getByTestId("save-manufacturer-btn"));

      await waitFor(() => {
        expect(api.updatePartManufacturer).toHaveBeenCalledWith(
          "IC-001",
          1,
          expect.objectContaining({
            mpn: "TPS54560BDDAR",
          })
        );
        expect(toast.success).toHaveBeenCalledWith("Manufacturer updated successfully");
      });
    });

    it("should prevent setting all manufacturers to non-primary", async () => {
      render(<PartDetail />);

      await waitFor(() => {
        expect(screen.getByText("Texas Instruments")).toBeInTheDocument();
      });

      // Open edit dialog for primary manufacturer
      fireEvent.click(screen.getByTestId("edit-manufacturer-1"));

      await waitFor(() => {
        expect(screen.getByTestId("primary-checkbox")).toBeInTheDocument();
      });

      // Try to uncheck primary
      const primaryCheckbox = screen.getByTestId("primary-checkbox");
      fireEvent.click(primaryCheckbox);

      // Try to save
      fireEvent.click(screen.getByTestId("save-manufacturer-btn"));

      await waitFor(() => {
        expect(screen.getByText("At least one manufacturer must be primary")).toBeInTheDocument();
        expect(api.updatePartManufacturer).not.toHaveBeenCalled();
      });
    });
  });

  describe("Delete manufacturer", () => {
    it("should delete manufacturer successfully", async () => {
      vi.mocked(api.deletePartManufacturer).mockResolvedValue({
        message: "Manufacturer deleted successfully",
      });

      render(<PartDetail />);

      await waitFor(() => {
        expect(screen.getByText("Texas Instruments")).toBeInTheDocument();
      });

      // Open delete dialog
      fireEvent.click(screen.getByTestId("delete-manufacturer-2"));

      await waitFor(() => {
        const dialog = screen.getByTestId("delete-manufacturer-dialog");
        expect(dialog).toBeInTheDocument();
        expect(dialog).toHaveTextContent("Analog Devices");
      });

      // Confirm delete
      fireEvent.click(screen.getByTestId("confirm-delete-manufacturer"));

      await waitFor(() => {
        expect(api.deletePartManufacturer).toHaveBeenCalledWith("IC-001", 2);
        expect(toast.success).toHaveBeenCalledWith("Manufacturer deleted successfully");
      });
    });

    it("should handle delete error", async () => {
      vi.mocked(api.deletePartManufacturer).mockRejectedValue(
        new Error("Failed to delete manufacturer")
      );

      render(<PartDetail />);

      await waitFor(() => {
        expect(screen.getByText("Texas Instruments")).toBeInTheDocument();
      });

      // Open delete dialog
      fireEvent.click(screen.getByTestId("delete-manufacturer-1"));

      // Confirm delete
      fireEvent.click(screen.getByTestId("confirm-delete-manufacturer"));

      await waitFor(() => {
        expect(toast.error).toHaveBeenCalledWith("Failed to delete manufacturer");
      });
    });
  });

  describe("Autocomplete dropdown", () => {
    it("should filter manufacturers by search query", async () => {
      render(<PartDetail />);

      await waitFor(() => {
        expect(screen.getByText("Texas Instruments")).toBeInTheDocument();
      });

      // Open add dialog
      fireEvent.click(screen.getByTestId("add-manufacturer-btn"));

      await waitFor(() => {
        expect(screen.getByTestId("manufacturer-input")).toBeInTheDocument();
      });

      // Type "Analog" to filter
      const input = screen.getByTestId("manufacturer-input");
      fireEvent.change(input, { target: { value: "Analog" } });

      await waitFor(() => {
        expect(screen.getByTestId("manufacturer-dropdown")).toBeInTheDocument();
        expect(screen.getByTestId("manufacturer-option-2")).toBeInTheDocument();
        expect(screen.queryByTestId("manufacturer-option-1")).not.toBeInTheDocument();
      });
    });

    it("should show email in dropdown options", async () => {
      render(<PartDetail />);

      await waitFor(() => {
        expect(screen.getByText("Texas Instruments")).toBeInTheDocument();
      });

      // Open add dialog
      fireEvent.click(screen.getByTestId("add-manufacturer-btn"));

      await waitFor(() => {
        expect(screen.getByTestId("manufacturer-input")).toBeInTheDocument();
      });

      // Focus to show all options
      const input = screen.getByTestId("manufacturer-input");
      fireEvent.focus(input);

      await waitFor(() => {
        expect(screen.getByTestId("manufacturer-dropdown")).toBeInTheDocument();
        expect(screen.getByText("sales@ti.com")).toBeInTheDocument();
        expect(screen.getByText("info@analog.com")).toBeInTheDocument();
      });
    });
  });
});
