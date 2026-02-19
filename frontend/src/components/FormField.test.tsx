import { describe, it, expect } from "vitest";
import { render, screen } from "../test/test-utils";
import { FormField } from "./FormField";
import { Input } from "./ui/input";
import "../test/setup-a11y";
import { axe } from "../test/a11y-test-utils";

describe("FormField Accessibility", () => {
  it("should associate label with input via htmlFor", () => {
    render(
      <FormField label="Part Number" htmlFor="ipn">
        <Input id="ipn" />
      </FormField>
    );
    
    const input = screen.getByLabelText("Part Number");
    expect(input).toBeInTheDocument();
    expect(input).toHaveAttribute("id", "ipn");
  });

  it("should mark required fields with visual indicator", () => {
    render(
      <FormField label="Part Number" htmlFor="ipn" required>
        <Input id="ipn" />
      </FormField>
    );
    
    expect(screen.getByText("*")).toBeInTheDocument();
  });

  it("should announce errors with role=alert", () => {
    render(
      <FormField label="Part Number" htmlFor="ipn" error="Part number is required">
        <Input id="ipn" aria-invalid="true" />
      </FormField>
    );
    
    const errorMessage = screen.getByRole("alert");
    expect(errorMessage).toHaveTextContent("Part number is required");
  });

  it("should not have accessibility violations", async () => {
    const { container } = render(
      <FormField label="Test Field" htmlFor="test" error="Error message">
        <Input id="test" aria-invalid="true" />
      </FormField>
    );
    
    const results = await axe(container);
    expect(results).toHaveNoViolations();
  });

  it("should show description when no error", () => {
    render(
      <FormField 
        label="Part Number" 
        htmlFor="ipn" 
        description="Format: XXX-NNNNN"
      >
        <Input id="ipn" />
      </FormField>
    );
    
    expect(screen.getByText("Format: XXX-NNNNN")).toBeInTheDocument();
  });

  it("should hide description when error is present", () => {
    render(
      <FormField 
        label="Part Number" 
        htmlFor="ipn" 
        description="Format: XXX-NNNNN"
        error="Invalid format"
      >
        <Input id="ipn" />
      </FormField>
    );
    
    expect(screen.queryByText("Format: XXX-NNNNN")).not.toBeInTheDocument();
    expect(screen.getByText("Invalid format")).toBeInTheDocument();
  });
});
