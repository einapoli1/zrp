import { useState, useEffect } from "react";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { 
  Select, 
  SelectContent, 
  SelectItem, 
  SelectTrigger, 
  SelectValue 
} from "./ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "./ui/dialog";
import { Badge } from "./ui/badge";
import { 
  Search, 
  Filter, 
  X, 
  Plus, 
  Save, 
  Trash2, 
  ChevronDown, 
  ChevronUp,
  RotateCcw,
  Clock,
  Star,
  Zap
} from "lucide-react";
import { api } from "../lib/api";
import { toast } from "sonner";

export interface SearchFilter {
  field: string;
  operator: string;
  value: any;
  andOr?: string;
}

export interface SearchQuery {
  entity_type: string;
  filters: SearchFilter[];
  search_text?: string;
  sort_by?: string;
  sort_order?: string;
  limit?: number;
  offset?: number;
}

export interface SavedSearch {
  id: string;
  name: string;
  entity_type: string;
  filters: SearchFilter[];
  sort_by: string;
  sort_order: string;
  created_by: string;
  is_public: boolean;
}

export interface QuickFilter {
  id: string;
  name: string;
  entity_type: string;
  filters: SearchFilter[];
}

interface AdvancedSearchProps {
  entityType: string;
  onSearch: (query: SearchQuery) => void;
  availableFields: { field: string; label: string; type: string }[];
  initialFilters?: SearchFilter[];
}

const OPERATORS = [
  { value: "eq", label: "Equals" },
  { value: "ne", label: "Not Equals" },
  { value: "contains", label: "Contains" },
  { value: "startswith", label: "Starts With" },
  { value: "endswith", label: "Ends With" },
  { value: "gt", label: "Greater Than" },
  { value: "lt", label: "Less Than" },
  { value: "gte", label: "Greater or Equal" },
  { value: "lte", label: "Less or Equal" },
  { value: "in", label: "In List" },
  { value: "between", label: "Between" },
  { value: "isnull", label: "Is Null" },
  { value: "isnotnull", label: "Is Not Null" },
];

export function AdvancedSearch({ 
  entityType, 
  onSearch, 
  availableFields,
  initialFilters = []
}: AdvancedSearchProps) {
  const [expanded, setExpanded] = useState(false);
  const [filters, setFilters] = useState<SearchFilter[]>(initialFilters);
  const [searchText, setSearchText] = useState("");
  const [savedSearches, setSavedSearches] = useState<SavedSearch[]>([]);
  const [quickFilters, setQuickFilters] = useState<QuickFilter[]>([]);
  const [searchHistory, setSearchHistory] = useState<any[]>([]);
  const [saveDialogOpen, setSaveDialogOpen] = useState(false);
  const [savedSearchName, setSavedSearchName] = useState("");
  const [isPublic, setIsPublic] = useState(false);
  const [sortBy, setSortBy] = useState("");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("asc");

  useEffect(() => {
    loadSavedSearches();
    loadQuickFilters();
    loadSearchHistory();
  }, [entityType]);

  const loadSavedSearches = async () => {
    try {
      const response = await api.get(`/saved-searches?entity_type=${entityType}`);
      setSavedSearches(response.data || []);
    } catch (error) {
      console.error("Failed to load saved searches:", error);
    }
  };

  const loadQuickFilters = async () => {
    try {
      const response = await api.get(`/search/quick-filters?entity_type=${entityType}`);
      setQuickFilters(response.data || []);
    } catch (error) {
      console.error("Failed to load quick filters:", error);
    }
  };

  const loadSearchHistory = async () => {
    try {
      const response = await api.get(`/search/history?entity_type=${entityType}&limit=5`);
      setSearchHistory(response.data || []);
    } catch (error) {
      console.error("Failed to load search history:", error);
    }
  };

  const addFilter = () => {
    setFilters([...filters, { 
      field: availableFields[0]?.field || "", 
      operator: "eq", 
      value: "",
      andOr: "AND"
    }]);
  };

  const removeFilter = (index: number) => {
    setFilters(filters.filter((_, i) => i !== index));
  };

  const updateFilter = (index: number, updates: Partial<SearchFilter>) => {
    const newFilters = [...filters];
    newFilters[index] = { ...newFilters[index], ...updates };
    setFilters(newFilters);
  };

  const handleSearch = () => {
    const query: SearchQuery = {
      entity_type: entityType,
      filters,
      search_text: searchText,
      sort_by: sortBy,
      sort_order: sortOrder,
      limit: 50,
      offset: 0
    };
    onSearch(query);
  };

  const clearFilters = () => {
    setFilters([]);
    setSearchText("");
    setSortBy("");
    setSortOrder("asc");
  };

  const applyQuickFilter = (quickFilter: QuickFilter) => {
    setFilters(quickFilter.filters);
    handleSearch();
  };

  const applySavedSearch = (savedSearch: SavedSearch) => {
    setFilters(savedSearch.filters);
    setSortBy(savedSearch.sort_by);
    setSortOrder(savedSearch.sort_order as "asc" | "desc");
    handleSearch();
    toast.success(`Applied saved search: ${savedSearch.name}`);
  };

  const saveCurrentSearch = async () => {
    if (!savedSearchName.trim()) {
      toast.error("Please enter a name for the saved search");
      return;
    }

    try {
      const payload = {
        name: savedSearchName,
        entity_type: entityType,
        filters,
        sort_by: sortBy,
        sort_order: sortOrder,
        is_public: isPublic
      };

      await api.post("/saved-searches", payload);
      toast.success("Search saved successfully");
      setSaveDialogOpen(false);
      setSavedSearchName("");
      setIsPublic(false);
      loadSavedSearches();
    } catch (error) {
      toast.error("Failed to save search");
    }
  };

  const deleteSavedSearch = async (id: string) => {
    try {
      await api.delete(`/saved-searches?id=${id}`);
      toast.success("Saved search deleted");
      loadSavedSearches();
    } catch (error) {
      toast.error("Failed to delete saved search");
    }
  };

  const getFieldType = (field: string): string => {
    const fieldDef = availableFields.find(f => f.field === field);
    return fieldDef?.type || "text";
  };

  return (
    <div className="space-y-4">
      {/* Search Bar with Expansion Toggle */}
      <div className="flex gap-2">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search (use field:value for advanced operators)..."
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSearch()}
            className="pl-10"
          />
        </div>
        <Button
          variant="outline"
          onClick={() => setExpanded(!expanded)}
          className="gap-2"
        >
          <Filter className="h-4 w-4" />
          Advanced
          {expanded ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
        </Button>
        <Button onClick={handleSearch}>Search</Button>
      </div>

      {/* Active Filter Chips */}
      {filters.length > 0 && (
        <div className="flex flex-wrap gap-2 items-center">
          <span className="text-sm text-muted-foreground">Active Filters:</span>
          {filters.map((filter, idx) => (
            <Badge key={idx} variant="secondary" className="gap-1">
              <span>{filter.field} {filter.operator} {JSON.stringify(filter.value)}</span>
              <X 
                className="h-3 w-3 cursor-pointer" 
                onClick={() => removeFilter(idx)}
              />
            </Badge>
          ))}
          <Button
            variant="ghost"
            size="sm"
            onClick={clearFilters}
            className="gap-1 h-7"
          >
            <RotateCcw className="h-3 w-3" />
            Clear All
          </Button>
        </div>
      )}

      {/* Advanced Panel */}
      {expanded && (
        <div className="border rounded-lg p-4 space-y-4">
          {/* Quick Filters */}
          {quickFilters.length > 0 && (
            <div>
              <Label className="text-sm font-medium mb-2 flex items-center gap-2">
                <Zap className="h-4 w-4" />
                Quick Filters
              </Label>
              <div className="flex flex-wrap gap-2">
                {quickFilters.map((qf) => (
                  <Button
                    key={qf.id}
                    variant="outline"
                    size="sm"
                    onClick={() => applyQuickFilter(qf)}
                  >
                    {qf.name}
                  </Button>
                ))}
              </div>
            </div>
          )}

          {/* Saved Searches */}
          {savedSearches.length > 0 && (
            <div>
              <Label className="text-sm font-medium mb-2 flex items-center gap-2">
                <Star className="h-4 w-4" />
                Saved Searches
              </Label>
              <div className="flex flex-wrap gap-2">
                {savedSearches.map((ss) => (
                  <div key={ss.id} className="flex items-center gap-1">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => applySavedSearch(ss)}
                    >
                      {ss.name}
                      {ss.is_public && <span className="ml-1 text-xs">(shared)</span>}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-7 w-7 p-0"
                      onClick={() => deleteSavedSearch(ss.id)}
                    >
                      <Trash2 className="h-3 w-3" />
                    </Button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Search History */}
          {searchHistory.length > 0 && (
            <div>
              <Label className="text-sm font-medium mb-2 flex items-center gap-2">
                <Clock className="h-4 w-4" />
                Recent Searches
              </Label>
              <div className="text-xs text-muted-foreground space-y-1">
                {searchHistory.map((entry, idx) => (
                  <div key={idx} className="truncate">
                    {entry.search_text || "Advanced search"}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Filter Builder */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <Label className="text-sm font-medium">Filters</Label>
              <Button
                variant="outline"
                size="sm"
                onClick={addFilter}
                className="gap-1"
              >
                <Plus className="h-4 w-4" />
                Add Filter
              </Button>
            </div>

            <div className="space-y-2">
              {filters.map((filter, idx) => (
                <div key={idx} className="flex gap-2 items-start">
                  {idx > 0 && (
                    <Select
                      value={filters[idx - 1].andOr || "AND"}
                      onValueChange={(value) => updateFilter(idx - 1, { andOr: value })}
                    >
                      <SelectTrigger className="w-20">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="AND">AND</SelectItem>
                        <SelectItem value="OR">OR</SelectItem>
                      </SelectContent>
                    </Select>
                  )}
                  
                  <Select
                    value={filter.field}
                    onValueChange={(value) => updateFilter(idx, { field: value })}
                  >
                    <SelectTrigger className="w-48">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {availableFields.map((f) => (
                        <SelectItem key={f.field} value={f.field}>
                          {f.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>

                  <Select
                    value={filter.operator}
                    onValueChange={(value) => updateFilter(idx, { operator: value })}
                  >
                    <SelectTrigger className="w-40">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {OPERATORS.map((op) => (
                        <SelectItem key={op.value} value={op.value}>
                          {op.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>

                  {!["isnull", "isnotnull"].includes(filter.operator) && (
                    <Input
                      placeholder="Value (use * for wildcard)"
                      value={filter.value}
                      onChange={(e) => updateFilter(idx, { value: e.target.value })}
                      className="flex-1"
                    />
                  )}

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => removeFilter(idx)}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              ))}
            </div>
          </div>

          {/* Sort Controls */}
          <div className="flex gap-2 items-end">
            <div className="flex-1">
              <Label className="text-sm">Sort By</Label>
              <Select value={sortBy} onValueChange={setSortBy}>
                <SelectTrigger>
                  <SelectValue placeholder="Select field..." />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">None</SelectItem>
                  {availableFields.map((f) => (
                    <SelectItem key={f.field} value={f.field}>
                      {f.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="w-32">
              <Label className="text-sm">Order</Label>
              <Select value={sortOrder} onValueChange={(v) => setSortOrder(v as "asc" | "desc")}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="asc">Ascending</SelectItem>
                  <SelectItem value="desc">Descending</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Actions */}
          <div className="flex justify-between pt-2 border-t">
            <Button
              variant="outline"
              onClick={() => setSaveDialogOpen(true)}
              className="gap-2"
            >
              <Save className="h-4 w-4" />
              Save Search
            </Button>
            <div className="flex gap-2">
              <Button variant="outline" onClick={clearFilters}>
                Clear All
              </Button>
              <Button onClick={handleSearch}>Apply Filters</Button>
            </div>
          </div>
        </div>
      )}

      {/* Save Search Dialog */}
      <Dialog open={saveDialogOpen} onOpenChange={setSaveDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Save Search</DialogTitle>
            <DialogDescription>
              Save this search configuration to reuse later
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div>
              <Label htmlFor="search-name">Search Name</Label>
              <Input
                id="search-name"
                value={savedSearchName}
                onChange={(e) => setSavedSearchName(e.target.value)}
                placeholder="e.g., Open High Priority Items"
              />
            </div>
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="is-public"
                checked={isPublic}
                onChange={(e) => setIsPublic(e.target.checked)}
                className="rounded"
              />
              <Label htmlFor="is-public" className="cursor-pointer">
                Share with team (make public)
              </Label>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setSaveDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={saveCurrentSearch}>Save</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
