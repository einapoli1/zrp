import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "./ui/dialog";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { Checkbox } from "./ui/checkbox";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "./ui/select";
import { Pencil } from "lucide-react";

export interface BulkEditField {
  key: string;
  label: string;
  type: "text" | "select";
  options?: { value: string; label: string }[];
}

interface BulkEditDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  fields: BulkEditField[];
  selectedCount: number;
  onSubmit: (updates: Record<string, string>) => Promise<{ success: number; failed: number; errors?: string[] } | void>;
  title?: string;
}

export function BulkEditDialog({
  open,
  onOpenChange,
  fields,
  selectedCount,
  onSubmit,
  title = "Bulk Edit",
}: BulkEditDialogProps) {
  const [enabledFields, setEnabledFields] = useState<Set<string>>(new Set());
  const [values, setValues] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState<{ success: number; failed: number } | null>(null);

  const toggleField = (key: string) => {
    const next = new Set(enabledFields);
    if (next.has(key)) {
      next.delete(key);
    } else {
      next.add(key);
    }
    setEnabledFields(next);
  };

  const handleSubmit = async () => {
    const updates: Record<string, string> = {};
    for (const key of enabledFields) {
      if (values[key] !== undefined && values[key] !== "") {
        updates[key] = values[key];
      }
    }
    if (Object.keys(updates).length === 0) return;

    setSubmitting(true);
    try {
      await onSubmit(updates);
      setResult({ success: selectedCount, failed: 0 });
    } catch {
      setResult({ success: 0, failed: selectedCount });
    } finally {
      setSubmitting(false);
    }
  };

  const handleClose = () => {
    setEnabledFields(new Set());
    setValues({});
    setResult(null);
    onOpenChange(false);
  };

  const enabledCount = Array.from(enabledFields).filter(
    (k) => values[k] !== undefined && values[k] !== ""
  ).length;

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Pencil className="h-5 w-5" />
            {title}
          </DialogTitle>
        </DialogHeader>

        {result ? (
          <div className="py-4 text-center">
            <p className="text-lg font-semibold text-green-600">
              ✓ Updated {result.success} item{result.success !== 1 ? "s" : ""} successfully
            </p>
            {result.failed > 0 && (
              <p className="text-sm text-red-600 mt-1">
                {result.failed} item{result.failed !== 1 ? "s" : ""} failed
              </p>
            )}
            <Button className="mt-4" onClick={handleClose}>
              Done
            </Button>
          </div>
        ) : (
          <>
            <p className="text-sm text-muted-foreground">
              Updating <strong>{selectedCount}</strong> selected item{selectedCount !== 1 ? "s" : ""}. Check the fields you want to change:
            </p>

            <div className="space-y-4 my-4">
              {fields.map((field) => (
                <div key={field.key} className="flex items-start gap-3">
                  <Checkbox
                    checked={enabledFields.has(field.key)}
                    onCheckedChange={() => toggleField(field.key)}
                    className="mt-2"
                  />
                  <div className="flex-1">
                    <Label className="text-sm font-medium">{field.label}</Label>
                    {field.type === "select" && field.options ? (
                      <Select
                        disabled={!enabledFields.has(field.key)}
                        value={values[field.key] || ""}
                        onValueChange={(v) =>
                          setValues((prev) => ({ ...prev, [field.key]: v }))
                        }
                      >
                        <SelectTrigger className="mt-1">
                          <SelectValue placeholder={`Select ${field.label.toLowerCase()}`} />
                        </SelectTrigger>
                        <SelectContent>
                          {field.options.map((opt) => (
                            <SelectItem key={opt.value} value={opt.value}>
                              {opt.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    ) : (
                      <Input
                        disabled={!enabledFields.has(field.key)}
                        className="mt-1"
                        placeholder={`New ${field.label.toLowerCase()}`}
                        value={values[field.key] || ""}
                        onChange={(e) =>
                          setValues((prev) => ({
                            ...prev,
                            [field.key]: e.target.value,
                          }))
                        }
                      />
                    )}
                  </div>
                </div>
              ))}
            </div>

            <DialogFooter>
              <Button variant="outline" onClick={handleClose}>
                Cancel
              </Button>
              <Button
                onClick={handleSubmit}
                disabled={submitting || enabledCount === 0}
              >
                {submitting ? "Updating..." : `Update ${selectedCount} Item${selectedCount !== 1 ? "s" : ""}`}
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
