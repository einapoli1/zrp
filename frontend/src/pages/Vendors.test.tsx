import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "../test/test-utils";
import { Vendors } from "./Vendors";
import { api, type Manufacturer } from "../lib/api";
import { toast } from "sonner";

vi.mock("../lib/api");
vi.mock("sonner");

const mockManufacturers: Manufacturer[] = [
  {
    id: 1,
    name: "Texas Instruments",
    contact_name: "John Doe",
    contact_email: "john@ti.com",
    contact_phone: "+1-555-123-4567",
    website: "https://ti.com",
    notes: "Primary supplier",
    approved: true,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
  },
  {
    id: 2,
    name: "Analog Devices",
    contact_name: "Jane Smith",
    contact_email: "jane@analog.com",
    contact_phone: "+1-555-987-6543",
    website: "https://analog.com",
    notes: "Secondary supplier",
    approved: false,
    created_at: "2024-01-02T00:00:00Z",
    updated_at: "2024-01-02T00:00:00Z",
  },
];

describe("Vendors", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("Render manufacturers list", () => {
    it("should render manufacturers table with data", async () => {
      vi.mocked(api.getManufacturers).mockResolvedValue(mockManufacturers);

      render(<Vendors />);

      await waitFor(() => {
        expect(screen.getByText("Texas Instruments")).toBeInTheDocument();
        expect(screen.getByText("Analog Devices")).toBeInTheDocument();
        expect(screen.getByText("John Doe")).toBeInTheDocument();
        expect(screen.getByText("Jane Smith")).toBeInTheDocument();
        expect(screen.getByText("john@ti.com")).toBeInTheDocument();
        expect(screen.getByText("jane@analog.com")).toBeInTheDocument();
      });

      expect(screen.getByTestId("approved-badge-1")).toBeInTheDocument();
      expect(screen.getByTestId("unapproved-badge-2")).toBeInTheDocument();
    });

    it("should show empty state when no manufacturers exist", async () => {
      vi.mocked(api.getManufacturers).mockResolvedValue([]);

      render(<Vendors />);

      await waitFor(() => {
        expect(screen.getByText("No manufacturers added")).toBeInTheDocument();
        expect(screen.getByText("Get started by adding your first manufacturer")).toBeInTheDocument();
      });
    });
  });

  describe("Add manufacturer", () => {
    it("should add a new manufacturer successfully", async () => {
      vi.mocked(api.getManufacturers).mockResolvedValue(mockManufacturers);
      vi.mocked(api.createManufacturer).mockResolvedValue({
        id: 3,
        name: "Microchip",
        approved: true,
        created_at: "2024-01-03T00:00:00Z",
        updated_at: "2024-01-03T00:00:00Z",
      });

      render(<Vendors />);

      await waitFor(() => {
        expect(screen.getByText("Texas Instruments")).toBeInTheDocument();
      });

      // Open add dialog
      fireEvent.click(screen.getByTestId("add-manufacturer-btn"));

      await waitFor(() => {
        expect(screen.getByTestId("manufacturer-dialog")).toBeInTheDocument();
        expect(screen.getByRole("heading", { name: "Add Manufacturer" })).toBeInTheDocument();
      });

      // Fill form
      fireEvent.change(screen.getByTestId("name-input"), {
        target: { value: "Microchip" },
      });
      fireEvent.change(screen.getByTestId("contact-name-input"), {
        target: { value: "Bob Johnson" },
      });
      fireEvent.change(screen.getByTestId("contact-email-input"), {
        target: { value: "bob@microchip.com" },
      });

      // Submit
      fireEvent.click(screen.getByTestId("save-btn"));

      await waitFor(() => {
        expect(api.createManufacturer).toHaveBeenCalledWith({
          name: "Microchip",
          contact_name: "Bob Johnson",
          contact_email: "bob@microchip.com",
          contact_phone: "",
          website: "",
          notes: "",
          approved: true,
        });
        expect(toast.success).toHaveBeenCalledWith("Manufacturer added successfully");
      });
    });

    it("should show fuzzy match suggestion for similar manufacturer name", async () => {
      vi.mocked(api.getManufacturers).mockResolvedValue(mockManufacturers);
      vi.mocked(api.createManufacturer).mockRejectedValue(
        new Error("A similar manufacturer 'Texas Instruments' already exists. Did you mean that one?")
      );

      render(<Vendors />);

      await waitFor(() => {
        expect(screen.getByText("Texas Instruments")).toBeInTheDocument();
      });

      // Open add dialog
      fireEvent.click(screen.getByTestId("add-manufacturer-btn"));

      // Fill form with similar name
      fireEvent.change(screen.getByTestId("name-input"), {
        target: { value: "Texas Instrumants" }, // Typo
      });

      // Submit
      fireEvent.click(screen.getByTestId("save-btn"));

      await waitFor(() => {
        expect(toast.error).toHaveBeenCalledWith(
          expect.stringContaining("similar"),
          expect.objectContaining({ duration: 6000 })
        );
      });
    });
  });

  describe("Edit manufacturer", () => {
    it("should edit an existing manufacturer successfully", async () => {
      vi.mocked(api.getManufacturers).mockResolvedValue(mockManufacturers);
      vi.mocked(api.updateManufacturer).mockResolvedValue({
        ...mockManufacturers[0],
        contact_name: "John Updated",
      });

      render(<Vendors />);

      await waitFor(() => {
        expect(screen.getByText("Texas Instruments")).toBeInTheDocument();
      });

      // Open edit dialog
      fireEvent.click(screen.getByTestId("edit-manufacturer-1"));

      await waitFor(() => {
        expect(screen.getByTestId("manufacturer-dialog")).toBeInTheDocument();
        expect(screen.getByText("Edit Manufacturer")).toBeInTheDocument();
      });

      // Update contact name
      const nameInput = screen.getByTestId("contact-name-input");
      fireEvent.change(nameInput, {
        target: { value: "John Updated" },
      });

      // Submit
      fireEvent.click(screen.getByTestId("save-btn"));

      await waitFor(() => {
        expect(api.updateManufacturer).toHaveBeenCalledWith(1, expect.objectContaining({
          contact_name: "John Updated",
        }));
        expect(toast.success).toHaveBeenCalledWith("Manufacturer updated successfully");
      });
    });

    it("should validate email format when editing", async () => {
      vi.mocked(api.getManufacturers).mockResolvedValue(mockManufacturers);

      render(<Vendors />);

      await waitFor(() => {
        expect(screen.getByText("Texas Instruments")).toBeInTheDocument();
      });

      // Open edit dialog
      fireEvent.click(screen.getByTestId("edit-manufacturer-1"));

      // Enter invalid email
      const emailInput = screen.getByTestId("contact-email-input");
      fireEvent.change(emailInput, {
        target: { value: "invalid-email" },
      });

      // Submit
      fireEvent.click(screen.getByTestId("save-btn"));

      await waitFor(() => {
        expect(screen.getByText("Invalid email address")).toBeInTheDocument();
        expect(api.updateManufacturer).not.toHaveBeenCalled();
      });
    });
  });

  describe("Delete manufacturer", () => {
    it("should delete a manufacturer successfully", async () => {
      vi.mocked(api.getManufacturers).mockResolvedValue(mockManufacturers);
      vi.mocked(api.deleteManufacturer).mockResolvedValue();

      render(<Vendors />);

      await waitFor(() => {
        expect(screen.getByText("Texas Instruments")).toBeInTheDocument();
      });

      // Open delete dialog
      fireEvent.click(screen.getByTestId("delete-manufacturer-1"));

      await waitFor(() => {
        expect(screen.getByTestId("delete-dialog")).toBeInTheDocument();
        expect(screen.getByText(/Delete Manufacturer/)).toBeInTheDocument();
      });

      // Confirm delete
      fireEvent.click(screen.getByTestId("confirm-delete-btn"));

      await waitFor(() => {
        expect(api.deleteManufacturer).toHaveBeenCalledWith(1);
        expect(toast.success).toHaveBeenCalledWith("Manufacturer deleted successfully");
      });
    });

    it("should show error when parts reference the manufacturer", async () => {
      vi.mocked(api.getManufacturers).mockResolvedValue(mockManufacturers);
      vi.mocked(api.deleteManufacturer).mockRejectedValue(
        new Error("Cannot delete manufacturer - 5 parts reference this manufacturer")
      );

      render(<Vendors />);

      await waitFor(() => {
        expect(screen.getByText("Texas Instruments")).toBeInTheDocument();
      });

      // Open delete dialog
      fireEvent.click(screen.getByTestId("delete-manufacturer-1"));

      // Confirm delete
      fireEvent.click(screen.getByTestId("confirm-delete-btn"));

      await waitFor(() => {
        expect(toast.error).toHaveBeenCalledWith(
          "Cannot delete: This manufacturer is referenced by existing parts",
          expect.objectContaining({ duration: 5000 })
        );
      });
    });
  });

  describe("Search/filter", () => {
    it("should filter manufacturers by search query", async () => {
      vi.mocked(api.getManufacturers).mockResolvedValue(mockManufacturers);

      render(<Vendors />);

      await waitFor(() => {
        expect(screen.getByText("Texas Instruments")).toBeInTheDocument();
        expect(screen.getByText("Analog Devices")).toBeInTheDocument();
      });

      // Search for "Texas"
      const searchInput = screen.getByTestId("search-input");
      fireEvent.change(searchInput, {
        target: { value: "Texas" },
      });

      await waitFor(() => {
        expect(screen.getByText("Texas Instruments")).toBeInTheDocument();
        expect(screen.queryByText("Analog Devices")).not.toBeInTheDocument();
      });

      // Clear search
      fireEvent.change(searchInput, {
        target: { value: "" },
      });

      await waitFor(() => {
        expect(screen.getByText("Texas Instruments")).toBeInTheDocument();
        expect(screen.getByText("Analog Devices")).toBeInTheDocument();
      });
    });
  });
});
