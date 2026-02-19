import { Loader2 } from "lucide-react";
import { Skeleton } from "./ui/skeleton";
import { cn } from "../lib/utils";

interface LoadingStateProps {
  /** Loading variant - spinner (centered), skeleton (list), or table (table rows) */
  variant?: "spinner" | "skeleton" | "table";
  /** Optional loading message */
  message?: string;
  /** Number of skeleton rows to show (for skeleton/table variants) */
  rows?: number;
  /** Additional className */
  className?: string;
}

/**
 * Standardized loading state component
 * 
 * @example
 * // Centered spinner with message
 * <LoadingState variant="spinner" message="Loading parts..." />
 * 
 * @example
 * // List skeleton
 * <LoadingState variant="skeleton" rows={5} />
 * 
 * @example
 * // Table skeleton
 * <LoadingState variant="table" rows={10} />
 */
export function LoadingState({ 
  variant = "spinner", 
  message = "Loading...",
  rows = 5,
  className 
}: LoadingStateProps) {
  if (variant === "spinner") {
    return (
      <div className={cn("flex items-center justify-center min-h-[400px]", className)}>
        <div className="text-center">
          <Loader2 className="h-8 w-8 animate-spin text-primary mx-auto" />
          {message && (
            <p className="mt-2 text-sm text-muted-foreground">{message}</p>
          )}
        </div>
      </div>
    );
  }

  if (variant === "table") {
    return (
      <div className={cn("space-y-2", className)}>
        {Array.from({ length: rows }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    );
  }

  // skeleton variant
  return (
    <div className={cn("space-y-4", className)}>
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="flex items-center space-x-4">
          <Skeleton className="h-12 w-12 rounded-full" />
          <div className="space-y-2 flex-1">
            <Skeleton className="h-4 w-[250px]" />
            <Skeleton className="h-4 w-[200px]" />
          </div>
        </div>
      ))}
    </div>
  );
}
