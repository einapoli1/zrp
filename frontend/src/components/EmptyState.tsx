import { Inbox } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { cn } from "../lib/utils";
import type { ReactNode } from "react";

interface EmptyStateProps {
  /** Icon to display (defaults to Inbox) */
  icon?: LucideIcon;
  /** Main title */
  title: string;
  /** Optional description */
  description?: string;
  /** Optional action button or element */
  action?: ReactNode;
  /** Additional className */
  className?: string;
}

/**
 * Standardized empty state component for lists and data views
 * 
 * @example
 * <EmptyState 
 *   icon={Package}
 *   title="No parts found"
 *   description="Get started by adding your first part"
 *   action={<Button onClick={handleCreate}>Add Part</Button>}
 * />
 * 
 * @example
 * // With search/filter context
 * <EmptyState 
 *   title="No results found"
 *   description="Try adjusting your search or filters"
 * />
 */
export function EmptyState({
  icon: Icon = Inbox,
  title,
  description,
  action,
  className
}: EmptyStateProps) {
  return (
    <div className={cn(
      "flex flex-col items-center justify-center py-12 px-4 text-center",
      className
    )}>
      <div className="rounded-full bg-muted p-3 mb-4">
        <Icon className="h-8 w-8 text-muted-foreground" />
      </div>
      <h3 className="text-lg font-semibold mb-1">{title}</h3>
      {description && (
        <p className="text-sm text-muted-foreground mb-4 max-w-md">
          {description}
        </p>
      )}
      {action && <div className="mt-2">{action}</div>}
    </div>
  );
}
