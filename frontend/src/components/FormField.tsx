import type { ReactNode } from "react";
import { Label } from "./ui/label";
import { cn } from "../lib/utils";

interface FormFieldProps {
  /** Field label */
  label: string;
  /** Field ID for label association */
  htmlFor?: string;
  /** Whether field is required */
  required?: boolean;
  /** Validation error message */
  error?: string;
  /** Helper text */
  description?: string;
  /** Form input element (Input, Select, Textarea, etc.) */
  children: ReactNode;
  /** Additional className for wrapper */
  className?: string;
}

/**
 * Standardized form field wrapper with label, validation, and spacing
 * 
 * @example
 * <FormField 
 *   label="Part Number" 
 *   htmlFor="ipn"
 *   required
 *   error={errors.ipn?.message}
 *   description="Use format: XXX-NNNNN"
 * >
 *   <Input id="ipn" {...register("ipn")} />
 * </FormField>
 */
export function FormField({
  label,
  htmlFor,
  required,
  error,
  description,
  children,
  className
}: FormFieldProps) {
  return (
    <div className={cn("space-y-2", className)}>
      <Label htmlFor={htmlFor} className="flex items-center gap-1">
        {label}
        {required && <span className="text-destructive">*</span>}
      </Label>
      {children}
      {description && !error && (
        <p className="text-xs text-muted-foreground">{description}</p>
      )}
      {error && (
        <p className="text-xs text-destructive" role="alert">
          {error}
        </p>
      )}
    </div>
  );
}
