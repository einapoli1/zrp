import { AlertCircle, RefreshCw } from "lucide-react";
import { Button } from "./ui/button";
import { cn } from "../lib/utils";

interface ErrorStateProps {
  /** Error title */
  title?: string;
  /** Error message/description */
  message?: string;
  /** Optional retry callback */
  onRetry?: () => void;
  /** Additional className */
  className?: string;
  /** Variant - inline (compact) or full (centered with padding) */
  variant?: "inline" | "full";
}

/**
 * Standardized error state component
 * 
 * @example
 * // Full error page
 * <ErrorState 
 *   title="Failed to load parts"
 *   message="Unable to connect to the server"
 *   onRetry={fetchParts}
 * />
 * 
 * @example
 * // Inline error (e.g., in a card)
 * <ErrorState 
 *   variant="inline"
 *   message="Failed to save changes"
 * />
 */
export function ErrorState({
  title = "Something went wrong",
  message = "An error occurred while loading this data",
  onRetry,
  className,
  variant = "full"
}: ErrorStateProps) {
  if (variant === "inline") {
    return (
      <div className={cn(
        "flex items-center gap-2 p-3 text-sm border border-destructive/50 bg-destructive/10 rounded-md",
        className
      )}>
        <AlertCircle className="h-4 w-4 text-destructive flex-shrink-0" />
        <div className="flex-1 min-w-0">
          {title && <div className="font-medium text-destructive">{title}</div>}
          <div className="text-destructive/90">{message}</div>
        </div>
        {onRetry && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onRetry}
            className="flex-shrink-0"
          >
            <RefreshCw className="h-3 w-3" />
          </Button>
        )}
      </div>
    );
  }

  return (
    <div className={cn(
      "flex flex-col items-center justify-center py-12 px-4 text-center",
      className
    )}>
      <div className="rounded-full bg-destructive/10 p-3 mb-4">
        <AlertCircle className="h-8 w-8 text-destructive" />
      </div>
      <h3 className="text-lg font-semibold mb-1">{title}</h3>
      <p className="text-sm text-muted-foreground mb-4 max-w-md">{message}</p>
      {onRetry && (
        <Button onClick={onRetry} variant="outline">
          <RefreshCw className="h-4 w-4 mr-2" />
          Try Again
        </Button>
      )}
    </div>
  );
}
