import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import userEvent from "@testing-library/user-event";
import { BulkEditDialog, type BulkEditField } from "./BulkEditDialog";

const mockOnSubmit = vi.fn();
const mockOnOpenChange = vi.fn();

const testFields: BulkEditField[] = [
  { key: "location", label: "Location", type: "text" },
  {
    key: "status",
    label: "Status",
    type: "select",
    options: [
      { value: "active", label: "Active" },
      { value: "inactive", label: "Inactive" },
    ],
  },
];

beforeEach(() => {
  vi.clearAllMocks();
  mockOnSubmit.mockResolvedValue(undefined);
});

describe("BulkEditDialog", () => {
  it("renders with correct selected count", () => {
    render(
      <BulkEditDialog
        open={true}
        onOpenChange={mockOnOpenChange}
        fields={testFields}
        selectedCount={5}
        onSubmit={mockOnSubmit}
        title="Bulk Edit Test"
      />
    );

    expect(screen.getByText("5")).toBeInTheDocument();
    expect(screen.getByText("Bulk Edit Test")).toBeInTheDocument();
    expect(screen.getByText("Location")).toBeInTheDocument();
    expect(screen.getByText("Status")).toBeInTheDocument();
  });

  it("submit button is disabled when no fields are enabled", () => {
    render(
      <BulkEditDialog
        open={true}
        onOpenChange={mockOnOpenChange}
        fields={testFields}
        selectedCount={3}
        onSubmit={mockOnSubmit}
      />
    );

    const submitBtn = screen.getByRole("button", { name: /Update 3 Items/i });
    expect(submitBtn).toBeDisabled();
  });

  it("enables field input when checkbox is checked", async () => {
    const user = userEvent.setup();

    render(
      <BulkEditDialog
        open={true}
        onOpenChange={mockOnOpenChange}
        fields={testFields}
        selectedCount={2}
        onSubmit={mockOnSubmit}
      />
    );

    // The Location input should be disabled initially
    const locationInput = screen.getByPlaceholderText("New location");
    expect(locationInput).toBeDisabled();

    // Check the first checkbox (Location)
    const checkboxes = screen.getAllByRole("checkbox");
    await user.click(checkboxes[0]);

    // Now input should be enabled
    expect(locationInput).not.toBeDisabled();
  });

  it("calls onSubmit with correct updates when submitted", async () => {
    const user = userEvent.setup();

    render(
      <BulkEditDialog
        open={true}
        onOpenChange={mockOnOpenChange}
        fields={testFields}
        selectedCount={2}
        onSubmit={mockOnSubmit}
      />
    );

    // Enable location field
    const checkboxes = screen.getAllByRole("checkbox");
    await user.click(checkboxes[0]);

    // Type a value
    const locationInput = screen.getByPlaceholderText("New location");
    await user.type(locationInput, "Shelf-B3");

    // Submit
    const submitBtn = screen.getByRole("button", { name: /Update 2 Items/i });
    await user.click(submitBtn);

    await waitFor(() => {
      expect(mockOnSubmit).toHaveBeenCalledWith({ location: "Shelf-B3" });
    });
  });

  it("shows success message after submit", async () => {
    const user = userEvent.setup();

    render(
      <BulkEditDialog
        open={true}
        onOpenChange={mockOnOpenChange}
        fields={testFields}
        selectedCount={2}
        onSubmit={mockOnSubmit}
      />
    );

    const checkboxes = screen.getAllByRole("checkbox");
    await user.click(checkboxes[0]);
    const locationInput = screen.getByPlaceholderText("New location");
    await user.type(locationInput, "X");

    const submitBtn = screen.getByRole("button", { name: /Update 2 Items/i });
    await user.click(submitBtn);

    await waitFor(() => {
      expect(screen.getByText(/Updated 2 items successfully/)).toBeInTheDocument();
    });
  });

  it("does not render when open is false", () => {
    render(
      <BulkEditDialog
        open={false}
        onOpenChange={mockOnOpenChange}
        fields={testFields}
        selectedCount={1}
        onSubmit={mockOnSubmit}
      />
    );

    expect(screen.queryByText("Bulk Edit")).not.toBeInTheDocument();
  });
});
