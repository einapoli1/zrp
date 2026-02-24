import { useEffect, useState, useCallback, useRef, useLayoutEffect } from "react";
import { createPortal } from "react-dom";
import { useParams, useNavigate, Link } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Skeleton } from "../components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../components/ui/table";
import {
  ArrowLeft,
  Layers,
  Plus,
  Trash2,
  Save,
  RefreshCw,
  AlertCircle,
  Loader2,
} from "lucide-react";
import { toast } from "sonner";
import { api, type BOMLine, type Part } from "../lib/api";

// ─── Row model ────────────────────────────────────────────────────────────────

interface BOMRow extends BOMLine {
  _id: number;
}

let _rowCounter = 0;
function newRow(partial: Partial<BOMLine> = {}): BOMRow {
  return {
    _id: ++_rowCounter,
    ipn: partial.ipn ?? "",
    qty: partial.qty ?? 1,
    ref: partial.ref ?? "",
    description: partial.description ?? "",
  };
}

// ─── IPN autocomplete cell ────────────────────────────────────────────────────

interface IPNAutocompleteProps {
  value: string;
  error?: string;
  onChange: (value: string) => void;
  onSelect: (part: Part) => void;
}

function IPNAutocomplete({ value, error, onChange, onSelect }: IPNAutocompleteProps) {
  const [suggestions, setSuggestions] = useState<Part[]>([]);
  const [open, setOpen] = useState(false);
  const [searching, setSearching] = useState(false);
  const [activeIdx, setActiveIdx] = useState(0);
  const [dropdownStyle, setDropdownStyle] = useState<React.CSSProperties>({});
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Reposition the portal dropdown to sit directly below the input
  const updateDropdownPosition = useCallback(() => {
    if (!inputRef.current) return;
    const rect = inputRef.current.getBoundingClientRect();
    setDropdownStyle({
      position: "fixed",
      top: rect.bottom + 4,
      left: rect.left,
      width: Math.max(rect.width, 320),
      zIndex: 9999,
    });
  }, []);

  useLayoutEffect(() => {
    if (open) updateDropdownPosition();
  }, [open, updateDropdownPosition]);

  // Keep position in sync while the user scrolls
  useEffect(() => {
    if (!open) return;
    const onScroll = () => updateDropdownPosition();
    window.addEventListener("scroll", onScroll, true);
    return () => window.removeEventListener("scroll", onScroll, true);
  }, [open, updateDropdownPosition]);

  // Debounced search: fires 200 ms after the user stops typing
  const search = useCallback((q: string) => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    if (!q.trim()) {
      setSuggestions([]);
      setOpen(false);
      return;
    }
    debounceRef.current = setTimeout(async () => {
      setSearching(true);
      try {
        const res = await api.getParts({ q, limit: 10 });
        const list = res.data ?? [];
        setSuggestions(list);
        setOpen(list.length > 0);
        setActiveIdx(0);
      } catch {
        setSuggestions([]);
        setOpen(false);
      } finally {
        setSearching(false);
      }
    }, 200);
  }, []);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    onChange(e.target.value);
    search(e.target.value);
  };

  const handleSelect = (part: Part) => {
    onSelect(part);
    setSuggestions([]);
    setOpen(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (!open || suggestions.length === 0) return;
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setActiveIdx((i) => Math.min(i + 1, suggestions.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActiveIdx((i) => Math.max(i - 1, 0));
    } else if (e.key === "Enter" || e.key === "Tab") {
      if (e.key === "Enter") e.preventDefault();
      handleSelect(suggestions[activeIdx]);
    } else if (e.key === "Escape") {
      setOpen(false);
    }
  };

  // Close dropdown when clicking outside
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (inputRef.current && !inputRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  const getDesc = (p: Part) =>
    p.fields?.description || p.fields?.desc || p.description || "";

  const dropdown = open && suggestions.length > 0 && createPortal(
    <div
      style={dropdownStyle}
      className="rounded-md border bg-popover shadow-lg overflow-hidden"
    >
      <ul className="max-h-52 overflow-y-auto py-1 text-sm">
        {suggestions.map((part, idx) => (
          <li
            key={part.ipn}
            className={`flex flex-col px-3 py-2 cursor-pointer select-none ${
              idx === activeIdx ? "bg-accent text-accent-foreground" : "hover:bg-muted"
            }`}
            onMouseEnter={() => setActiveIdx(idx)}
            onMouseDown={(e) => {
              e.preventDefault();
              handleSelect(part);
            }}
          >
            <span className="font-mono font-medium leading-tight">{part.ipn}</span>
            {getDesc(part) && (
              <span className="text-xs text-muted-foreground truncate">
                {getDesc(part)}
              </span>
            )}
          </li>
        ))}
      </ul>
    </div>,
    document.body
  );

  return (
    <div className="space-y-1">
      <div className="relative">
        <Input
          ref={inputRef}
          value={value}
          placeholder="e.g. RES-001"
          className={`font-mono text-sm pr-7 ${error ? "border-destructive" : ""}`}
          autoComplete="off"
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          onFocus={() => {
            if (suggestions.length > 0) {
              updateDropdownPosition();
              setOpen(true);
            }
          }}
        />
        {searching && (
          <Loader2 className="absolute right-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 animate-spin text-muted-foreground" />
        )}
      </div>
      {error && <p className="text-xs text-destructive">{error}</p>}
      {dropdown}
    </div>
  );
}

// ─── Main page ────────────────────────────────────────────────────────────────

function BOMCreator() {
  const { ipn } = useParams<{ ipn: string }>();
  const navigate = useNavigate();

  const [part, setPart] = useState<Part | null>(null);
  const [rows, setRows] = useState<BOMRow[]>([newRow()]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [rowErrors, setRowErrors] = useState<Record<number, string>>({});

  const decodedIPN = ipn ? decodeURIComponent(ipn) : "";
  const isAssembly =
    decodedIPN.toUpperCase().startsWith("ASY-") ||
    decodedIPN.toUpperCase().startsWith("PCA-");

  useEffect(() => {
    if (!decodedIPN) return;
    loadData();
  }, [decodedIPN]);

  const loadData = async () => {
    setLoading(true);
    try {
      const partData = await api.getPart(decodedIPN);
      setPart(partData);

      try {
        const bomData = await api.getPartBOM(decodedIPN);
        if (bomData.children && bomData.children.length > 0) {
          setRows(
            bomData.children.map((child) =>
              newRow({
                ipn: child.ipn,
                qty: child.qty ?? 1,
                ref: child.ref ?? "",
                description: child.description ?? "",
              })
            )
          );
        }
      } catch {
        setRows([newRow()]);
      }
    } catch {
      toast.error("Failed to load part");
    } finally {
      setLoading(false);
    }
  };

  const addRow = () => setRows((prev) => [...prev, newRow()]);

  const removeRow = (id: number) => {
    setRows((prev) => prev.filter((r) => r._id !== id));
    setRowErrors((prev) => {
      const next = { ...prev };
      delete next[id];
      return next;
    });
  };

  const updateRow = (id: number, field: keyof BOMLine, value: string | number) => {
    setRows((prev) =>
      prev.map((r) => (r._id === id ? { ...r, [field]: value } : r))
    );
    if (field === "ipn") {
      setRowErrors((prev) => {
        const next = { ...prev };
        delete next[id];
        return next;
      });
    }
  };

  // Called when user selects a suggestion — fills IPN + description
  const handleIPNSelect = (id: number, selected: Part) => {
    const desc =
      selected.fields?.description ||
      selected.fields?.desc ||
      selected.description ||
      "";
    setRows((prev) =>
      prev.map((r) =>
        r._id === id
          ? { ...r, ipn: selected.ipn, description: r.description || desc }
          : r
      )
    );
    setRowErrors((prev) => {
      const next = { ...prev };
      delete next[id];
      return next;
    });
  };

  const validate = (): boolean => {
    const errors: Record<number, string> = {};
    for (const row of rows) {
      if (row.ipn.trim() && row.qty <= 0) {
        errors[row._id] = "Qty must be > 0";
      }
    }
    setRowErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSave = async () => {
    if (!validate()) {
      toast.error("Please fix errors before saving");
      return;
    }
    const linesToSave: BOMLine[] = rows
      .filter((r) => r.ipn.trim() !== "")
      .map(({ ipn, qty, ref, description }) => ({ ipn, qty, ref, description }));

    setSaving(true);
    try {
      await api.updatePartBOM(decodedIPN, linesToSave);
      toast.success(
        `BOM saved for ${decodedIPN} (${linesToSave.length} line${linesToSave.length !== 1 ? "s" : ""})`
      );
    } catch (err) {
      toast.error("Failed to save BOM");
      console.error(err);
    } finally {
      setSaving(false);
    }
  };

  if (!isAssembly && !loading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center space-x-4">
          <Button variant="ghost" onClick={() => navigate("/parts")}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Parts
          </Button>
        </div>
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <AlertCircle className="h-12 w-12 text-destructive mb-4" />
            <h3 className="text-lg font-semibold mb-2">Not an Assembly</h3>
            <p className="text-muted-foreground text-center">
              BOM Creator is only available for assembly parts (ASY- or PCA- prefix).
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center space-x-4">
          <Skeleton className="h-10 w-10" />
          <Skeleton className="h-8 w-64" />
        </div>
        <Skeleton className="h-96" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Button
            variant="ghost"
            onClick={() => navigate(`/parts/${encodeURIComponent(decodedIPN)}`)}
          >
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Part
          </Button>
          <div>
            <h1 className="text-3xl font-bold tracking-tight font-mono">
              {decodedIPN}
            </h1>
            <p className="text-muted-foreground">
              {part?.fields?.description || part?.description || "BOM Creator"}
            </p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          <Badge variant="secondary">BOM Creator</Badge>
          <Button
            variant="outline"
            size="sm"
            onClick={loadData}
            disabled={loading || saving}
          >
            <RefreshCw className="h-4 w-4 mr-2" />
            Reload
          </Button>
          <Button
            size="sm"
            onClick={handleSave}
            disabled={saving || rows.filter((r) => r.ipn.trim()).length === 0}
          >
            <Save className="h-4 w-4 mr-2" />
            {saving ? "Saving..." : "Save BOM"}
          </Button>
        </div>
      </div>

      {/* BOM Editor */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center">
              <Layers className="h-5 w-5 mr-2" />
              Bill of Materials
              <Badge variant="outline" className="ml-2">
                {rows.filter((r) => r.ipn.trim()).length} line
                {rows.filter((r) => r.ipn.trim()).length !== 1 ? "s" : ""}
              </Badge>
            </CardTitle>
            <Button variant="outline" size="sm" onClick={addRow}>
              <Plus className="h-4 w-4 mr-2" />
              Add Line
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[220px]">IPN</TableHead>
                <TableHead className="w-[80px]">Qty</TableHead>
                <TableHead className="w-[150px]">Reference</TableHead>
                <TableHead>Description</TableHead>
                <TableHead className="w-[50px]"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => (
                <TableRow key={row._id}>
                  <TableCell className="align-top pt-3">
                    <IPNAutocomplete
                      value={row.ipn}
                      error={rowErrors[row._id]}
                      onChange={(val) => updateRow(row._id, "ipn", val)}
                      onSelect={(part) => handleIPNSelect(row._id, part)}
                    />
                  </TableCell>
                  <TableCell className="align-top pt-3">
                    <Input
                      type="number"
                      min={0.001}
                      step="any"
                      value={row.qty}
                      className="text-sm"
                      onChange={(e) =>
                        updateRow(row._id, "qty", parseFloat(e.target.value) || 1)
                      }
                    />
                  </TableCell>
                  <TableCell className="align-top pt-3">
                    <Input
                      value={row.ref}
                      placeholder="e.g. R1,R2"
                      className="text-sm font-mono"
                      onChange={(e) => updateRow(row._id, "ref", e.target.value)}
                    />
                  </TableCell>
                  <TableCell className="align-top pt-3">
                    <Input
                      value={row.description}
                      placeholder="Description"
                      className="text-sm"
                      onChange={(e) =>
                        updateRow(row._id, "description", e.target.value)
                      }
                    />
                  </TableCell>
                  <TableCell className="align-top pt-3">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => removeRow(row._id)}
                      disabled={rows.length === 1}
                    >
                      <Trash2 className="h-4 w-4 text-muted-foreground hover:text-destructive" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          <div className="flex items-center justify-between mt-4 pt-4 border-t">
            <Button variant="outline" size="sm" onClick={addRow}>
              <Plus className="h-4 w-4 mr-2" />
              Add Line
            </Button>
            <div className="flex items-center space-x-2 text-sm text-muted-foreground">
              <span>
                Saved to{" "}
                <span className="font-mono">{decodedIPN}.csv</span>
              </span>
              <span>·</span>
              <Link
                to={`/parts/${encodeURIComponent(decodedIPN)}`}
                className="text-primary hover:underline"
              >
                View BOM →
              </Link>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

export default BOMCreator;
