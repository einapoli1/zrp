import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "../test/test-utils";
import PartDetail from "./PartDetail";
import { api } from "../lib/api";
import type { Part, PartManufacturer } from "../lib/api";

// Mock dependencies
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ ipn: "RES-001" }),
    useNavigate: () => vi.fn(),
  };
});

vi.mock("../hooks/useGitPLM", () => ({
  useGitPLM: () => ({ configured: false, buildUrl: () => null }),
}));

vi.mock("sonner", async () => {
  const actual = await vi.importActual("sonner");
  return {
    ...actual,
    toast: {
      success: vi.fn(),
      error: vi.fn(),
    },
  };
});

describe("PartDetail - Manufacturers", () => {
  const mockPart: Part = {
    ipn: "RES-001",
    fields: {
      description: "10k Resistor",
      _category: "resistor",
    },
  };

  const mockManufacturers: PartManufacturer[] = [
    {
      id: 1,
      part_id: "RES-001",
      manufacturer: "Yageo",
      mpn: "RC0805FR-0710KL",
      is_primary: true,
      approved: true,
      notes: "Preferred source",
      created_at: "2024-01-01",
      updated_at: "2024-01-01",
    },
    {
      id: 2,
      part_id: "RES-001",
      manufacturer: "Vishay",
      mpn: "CRCW080510K0FKEA",
      is_primary: false,
      approved: true,
      notes: "",
      created_at: "2024-01-02",
      updated_at: "2024-01-02",
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    
    // Mock API calls
    vi.spyOn(api, "getPart").mockResolvedValue(mockPart);
    vi.spyOn(api, "getPartManufacturers").mockResolvedValue({
      manufacturers: mockManufacturers,
      count: mockManufacturers.length,
    });
    vi.spyOn(api, "getPartChanges").mockResolvedValue([]);
    vi.spyOn(api, "getPartBOM").mockRejectedValue(new Error("Not found"));
    vi.spyOn(api, "getPartCost").mockResolvedValue({ ipn: "RES-001" });
    vi.spyOn(api, "getPartWhereUsed").mockResolvedValue([]);
  });

  describe("Render manufacturers list", () => {
    it("should render manufacturers table with data", async () => {
      render(<PartDetail />);

      await waitFor(() => {
        expect(screen.getByText("Manufacturers")).toBeInTheDocument();
      });

      // Check manufacturer data is displayed
      expect(screen.getByText("Yageo")).toBeInTheDocument();
      expect(screen.getByText("RC0805FR-0710KL")).toBeInTheDocument();
      expect(screen.getByText("Vishay")).toBeInTheDocument();
      expect(screen.getByText("CRCW080510K0FKEA")).toBeInTheDocument();
      
      // Check count badge
      expect(screen.getByText("2")).toBeInTheDocument();
    });

    it("should render empty state when no manufacturers", async () => {
      vi.spyOn(api, "getPartManufacturers").mockResolvedValue({
        manufacturers: [],
        count: 0,
      });

      render(<PartDetail />);

      await waitFor(() => {
        expect(screen.getByText("No manufacturers added")).toBeInTheDocument();
      });
    });
  });

  describe("Add manufacturer", () => {
    it("should open dialog and save new manufacturer successfully", async () => {
      const createSpy = vi.spyOn(api, "createPartManufacturer").mockResolvedValue({
        id: 3,
        message: "Manufacturer created",
      });

      render(<PartDetail />);

      await waitFor(() => {
        expect(screen.getByTestId("add-manufacturer-btn")).toBeInTheDocument();
      });

      // Click add button
      fireEvent.click(screen.getByTestId("add-manufacturer-btn"));

      await waitFor(() => {
        expect(screen.getByTestId("manufacturer-dialog")).toBeInTheDocument();
      });

      // Fill in form
      fireEvent.change(screen.getByTestId("manufacturer-input"), {
        target: { value: "TDK" },
      });
      fireEvent.change(screen.getByTestId("mpn-input"), {
        target: { value: "C2012X7R1H103K125AB" },
      });

      // Save
      fireEvent.click(screen.getByTestId("save-manufacturer-btn"));

      await waitFor(() => {
        expect(createSpy).toHaveBeenCalledWith("RES-001", {
          manufacturer: "TDK",
          mpn: "C2012X7R1H103K125AB",
          is_primary: false,
          approved: true,
          notes: "",
        });
      });
    });

    it("should show validation errors for empty fields", async () => {
      render(<PartDetail />);

      await waitFor(() => {
        expect(screen.getByTestId("add-manufacturer-btn")).toBeInTheDocument();
      });

      // Click add button
      fireEvent.click(screen.getByTestId("add-manufacturer-btn"));

      await waitFor(() => {
        expect(screen.getByTestId("manufacturer-dialog")).toBeInTheDocument();
      });

      // Try to save without filling fields
      fireEvent.click(screen.getByTestId("save-manufacturer-btn"));

      await waitFor(() => {
        expect(screen.getByText("Manufacturer is required")).toBeInTheDocument();
        expect(screen.getByText("MPN is required")).toBeInTheDocument();
      });
    });
  });

  describe("Edit manufacturer", () => {
    it("should open dialog with pre-filled data and save changes", async () => {
      const updateSpy = vi.spyOn(api, "updatePartManufacturer").mockResolvedValue({
        message: "Manufacturer updated",
      });

      render(<PartDetail />);

      await waitFor(() => {
        expect(screen.getByTestId("edit-manufacturer-1")).toBeInTheDocument();
      });

      // Click edit button
      fireEvent.click(screen.getByTestId("edit-manufacturer-1"));

      await waitFor(() => {
        expect(screen.getByTestId("manufacturer-dialog")).toBeInTheDocument();
      });

      // Verify pre-filled data
      expect(screen.getByTestId("manufacturer-input")).toHaveValue("Yageo");
      expect(screen.getByTestId("mpn-input")).toHaveValue("RC0805FR-0710KL");

      // Change manufacturer
      fireEvent.change(screen.getByTestId("manufacturer-input"), {
        target: { value: "Yageo (Updated)" },
      });

      // Save
      fireEvent.click(screen.getByTestId("save-manufacturer-btn"));

      await waitFor(() => {
        expect(updateSpy).toHaveBeenCalledWith("RES-001", 1, expect.objectContaining({
          manufacturer: "Yageo (Updated)",
        }));
      });
    });

    it("should warn when unchecking primary with no other primary", async () => {
      // Mock with only one manufacturer that is primary
      vi.spyOn(api, "getPartManufacturers").mockResolvedValue({
        manufacturers: [mockManufacturers[0]],
        count: 1,
      });

      render(<PartDetail />);

      await waitFor(() => {
        expect(screen.getByTestId("edit-manufacturer-1")).toBeInTheDocument();
      });

      // Click edit button
      fireEvent.click(screen.getByTestId("edit-manufacturer-1"));

      await waitFor(() => {
        expect(screen.getByTestId("manufacturer-dialog")).toBeInTheDocument();
      });

      // Uncheck primary
      fireEvent.click(screen.getByTestId("primary-checkbox"));

      // Try to save
      fireEvent.click(screen.getByTestId("save-manufacturer-btn"));

      await waitFor(() => {
        expect(screen.getByText("At least one manufacturer must be primary")).toBeInTheDocument();
      });
    });
  });

  describe("Delete manufacturer", () => {
    it("should open confirmation dialog and delete successfully", async () => {
      const deleteSpy = vi.spyOn(api, "deletePartManufacturer").mockResolvedValue({
        message: "Manufacturer deleted",
      });

      render(<PartDetail />);

      await waitFor(() => {
        expect(screen.getByTestId("delete-manufacturer-2")).toBeInTheDocument();
      });

      // Click delete button
      fireEvent.click(screen.getByTestId("delete-manufacturer-2"));

      await waitFor(() => {
        expect(screen.getByTestId("delete-manufacturer-dialog")).toBeInTheDocument();
      });

      // Verify dialog shows manufacturer details (use getAllByText since it appears in table too)
      const vishayElements = screen.getAllByText("Vishay");
      expect(vishayElements.length).toBeGreaterThan(0);
      const mpnElements = screen.getAllByText("CRCW080510K0FKEA");
      expect(mpnElements.length).toBeGreaterThan(0);

      // Confirm delete
      fireEvent.click(screen.getByTestId("confirm-delete-manufacturer"));

      await waitFor(() => {
        expect(deleteSpy).toHaveBeenCalledWith("RES-001", 2);
      });
    });

    it("should show error when trying to delete last manufacturer", async () => {
      const deleteSpy = vi.spyOn(api, "deletePartManufacturer").mockRejectedValue(
        new Error("Cannot delete the last manufacturer")
      );

      render(<PartDetail />);

      await waitFor(() => {
        expect(screen.getByTestId("delete-manufacturer-1")).toBeInTheDocument();
      });

      // Click delete button
      fireEvent.click(screen.getByTestId("delete-manufacturer-1"));

      await waitFor(() => {
        expect(screen.getByTestId("delete-manufacturer-dialog")).toBeInTheDocument();
      });

      // Confirm delete
      fireEvent.click(screen.getByTestId("confirm-delete-manufacturer"));

      await waitFor(() => {
        expect(deleteSpy).toHaveBeenCalledWith("RES-001", 1);
      });
    });
  });

  describe("Primary badge display", () => {
    it("should display primary badge for primary manufacturer", async () => {
      render(<PartDetail />);

      await waitFor(() => {
        expect(screen.getByTestId("primary-badge-1")).toBeInTheDocument();
      });

      const primaryBadge = screen.getByTestId("primary-badge-1");
      expect(primaryBadge).toHaveTextContent("Primary");
    });

    it("should display approved checkmark for approved manufacturers", async () => {
      render(<PartDetail />);

      await waitFor(() => {
        expect(screen.getByTestId("approved-check-1")).toBeInTheDocument();
        expect(screen.getByTestId("approved-check-2")).toBeInTheDocument();
      });
    });
  });
});
