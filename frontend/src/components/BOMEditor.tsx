import { useState, useEffect } from "react";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "./ui/table";
import { Trash2, Plus, Save, X } from "lucide-react";
import { api } from "../lib/api";
import { toast } from "sonner";

interface BOMEditorItem {
  id: string;
  ipn: string;
  description: string;
  quantity: number;
  ref_des: string;
}

interface BOMEditorProps {
  assemblyIPN: string;
  initialItems?: BOMEditorItem[];
  onSave?: () => void;
  onCancel?: () => void;
}

export function BOMEditor({ assemblyIPN, initialItems = [], onSave, onCancel }: BOMEditorProps) {
  const [items, setItems] = useState<BOMEditorItem[]>(initialItems.length > 0 ? initialItems : [
    { id: crypto.randomUUID(), ipn: "", description: "", quantity: 1, ref_des: "" }
  ]);
  const [saving, setSaving] = useState(false);
  const [searchResults, setSearchResults] = useState<Record<string, any[]>>({});
  const [activeSearch, setActiveSearch] = useState<string | null>(null);

  const addRow = () => {
    setItems([...items, { 
      id: crypto.randomUUID(), 
      ipn: "", 
      description: "", 
      quantity: 1, 
      ref_des: "" 
    }]);
  };

  const removeRow = (id: string) => {
    setItems(items.filter(item => item.id !== id));
  };

  const updateItem = (id: string, field: keyof BOMEditorItem, value: any) => {
    setItems(items.map(item => 
      item.id === id ? { ...item, [field]: value } : item
    ));
  };

  const searchParts = async (query: string, itemId: string) => {
    if (query.length < 2) {
      setSearchResults({ ...searchResults, [itemId]: [] });
      return;
    }

    try {
      const response = await api.getParts({ q: query, limit: 10 });
      setSearchResults({ ...searchResults, [itemId]: response.data });
    } catch (error) {
      console.error("Search error:", error);
    }
  };

  const selectPart = async (itemId: string, ipn: string) => {
    // Auto-fill description
    try {
      const part = await api.getPart(ipn);
      const description = part.fields?.description || part.fields?.desc || "";
      
      setItems(items.map(item => 
        item.id === itemId 
          ? { ...item, ipn, description } 
          : item
      ));
      
      // Clear search results
      setSearchResults({ ...searchResults, [itemId]: [] });
      setActiveSearch(null);
    } catch (error) {
      console.error("Failed to fetch part details:", error);
      toast.error("Failed to fetch part details");
    }
  };

  const validateAndSave = async () => {
    // Validate all IPNs are non-empty and exist
    const invalidItems = items.filter(item => item.ipn && item.ipn.trim() !== "");
    
    if (invalidItems.length === 0 && items.some(item => item.ipn.trim() === "")) {
      toast.error("Please remove empty rows or fill in part numbers");
      return;
    }

    // Filter out empty rows
    const validItems = items.filter(item => item.ipn && item.ipn.trim() !== "");

    setSaving(true);
    try {
      await api.saveBOM(
        assemblyIPN, 
        validItems.map(item => ({
          ipn: item.ipn,
          description: item.description,
          quantity: item.quantity,
          ref_des: item.ref_des,
        }))
      );
      toast.success("BOM saved successfully");
      if (onSave) onSave();
    } catch (error: any) {
      const errorMsg = error?.message || "Failed to save BOM";
      
      // Check if error mentions a specific part not found
      if (errorMsg.includes("not found")) {
        toast.error(`Validation Error: ${errorMsg}`);
      } else {
        toast.error(errorMsg);
      }
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h3 className="text-lg font-semibold">Edit BOM for {assemblyIPN}</h3>
        <div className="space-x-2">
          {onCancel && (
            <Button variant="outline" onClick={onCancel} disabled={saving}>
              <X className="h-4 w-4 mr-2" />
              Cancel
            </Button>
          )}
          <Button onClick={addRow} variant="outline" disabled={saving}>
            <Plus className="h-4 w-4 mr-2" />
            Add Row
          </Button>
          <Button onClick={validateAndSave} disabled={saving}>
            <Save className="h-4 w-4 mr-2" />
            {saving ? "Saving..." : "Save BOM"}
          </Button>
        </div>
      </div>

      <div className="border rounded-lg overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[200px]">IPN</TableHead>
              <TableHead className="w-[300px]">Description</TableHead>
              <TableHead className="w-[100px]">Quantity</TableHead>
              <TableHead className="w-[200px]">Ref Des</TableHead>
              <TableHead className="w-[60px]"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((item) => (
              <TableRow key={item.id}>
                <TableCell className="relative">
                  <Input
                    value={item.ipn}
                    onChange={(e) => {
                      updateItem(item.id, "ipn", e.target.value);
                      searchParts(e.target.value, item.id);
                      setActiveSearch(item.id);
                    }}
                    onFocus={() => {
                      if (item.ipn.length >= 2) {
                        searchParts(item.ipn, item.id);
                        setActiveSearch(item.id);
                      }
                    }}
                    onBlur={() => {
                      // Delay to allow click on dropdown
                      setTimeout(() => {
                        if (activeSearch === item.id) {
                          setActiveSearch(null);
                        }
                      }, 200);
                    }}
                    placeholder="Search IPN..."
                    className="font-mono"
                  />
                  {activeSearch === item.id && searchResults[item.id]?.length > 0 && (
                    <div className="absolute z-50 w-full mt-1 bg-white border rounded-md shadow-lg max-h-60 overflow-y-auto">
                      {searchResults[item.id].map((part: any) => (
                        <div
                          key={part.ipn}
                          className="px-3 py-2 hover:bg-gray-100 cursor-pointer"
                          onMouseDown={() => selectPart(item.id, part.ipn)}
                        >
                          <div className="font-mono text-sm font-medium">{part.ipn}</div>
                          <div className="text-xs text-muted-foreground">
                            {part.fields?.description || part.fields?.desc || ""}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </TableCell>
                <TableCell>
                  <Input
                    value={item.description}
                    onChange={(e) => updateItem(item.id, "description", e.target.value)}
                    placeholder="Auto-filled from part"
                  />
                </TableCell>
                <TableCell>
                  <Input
                    type="number"
                    min="1"
                    value={item.quantity}
                    onChange={(e) => updateItem(item.id, "quantity", parseFloat(e.target.value) || 1)}
                  />
                </TableCell>
                <TableCell>
                  <Input
                    value={item.ref_des}
                    onChange={(e) => updateItem(item.id, "ref_des", e.target.value)}
                    placeholder="R1, R2, R3"
                  />
                </TableCell>
                <TableCell>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => removeRow(item.id)}
                    disabled={items.length === 1}
                  >
                    <Trash2 className="h-4 w-4 text-destructive" />
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <div className="text-sm text-muted-foreground">
        <p>💡 Tip: Start typing an IPN to search and auto-fill the description</p>
        <p>⚠️ All IPNs must exist in the parts database before saving</p>
      </div>
    </div>
  );
}
