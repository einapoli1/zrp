import { useEffect, useState, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { 
  Select, 
  SelectContent, 
  SelectItem, 
  SelectTrigger, 
  SelectValue 
} from "../components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "../components/ui/dialog";
// Table components used by ConfigurableTable internally
import { Skeleton } from "../components/ui/skeleton";
import { 
  Search, 
  Filter,
  ScanLine,
  ChevronLeft,
  ChevronRight,
  RotateCcw,
  Plus
} from "lucide-react";
import { api, type Part, type Category, type ApiResponse } from "../lib/api";
import { ConfigurableTable, type ColumnDef } from "../components/ConfigurableTable";
import { BarcodeScanner } from "../components/BarcodeScanner";
import { useGitPLM } from "../hooks/useGitPLM";
import { ExternalLink } from "lucide-react";
import { toast } from "sonner";
interface PartWithFields extends Part {
  category?: string;
  description?: string;
  cost?: number;
  stock?: number;
  status?: string;
}

interface CreatePartData {
  ipn: string;
  category: string;
  dynamicFields: Record<string, string>;
}

interface NewCategoryData {
  title: string;
  prefix: string;
}

function Parts() {
  const navigate = useNavigate();
  const [parts, setParts] = useState<Part[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState("");
  const [showScanner, setShowScanner] = useState(false);
  const [selectedCategory, setSelectedCategory] = useState<string>("all");
  const [currentPage, setCurrentPage] = useState(1);
  const [totalParts, setTotalParts] = useState(0);
  const { configured: gitplmConfigured, buildUrl: gitplmUrl } = useGitPLM();
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState("");
  const [ipnError, setIpnError] = useState("");
  const [newCatDialogOpen, setNewCatDialogOpen] = useState(false);
  const [newCatData, setNewCatData] = useState<NewCategoryData>({ title: "", prefix: "" });
  const [creatingCategory, setCreatingCategory] = useState(false);
  const pageSize = 50;

  const [partForm, setPartForm] = useState<CreatePartData>({
    ipn: "",
    category: "",
    dynamicFields: {},
  });

  // Debounced search effect
  useEffect(() => {
    const timeoutId = setTimeout(() => {
      fetchParts();
    }, 300);
    return () => clearTimeout(timeoutId);
  }, [searchQuery, selectedCategory, currentPage]);

  // Load categories on mount
  useEffect(() => {
    fetchCategories();
  }, []);

  const fetchCategories = async () => {
    try {
      const data = await api.getCategories();
      setCategories(data);
    } catch (error) {
      toast.error("Failed to fetch categories"); console.error("Failed to fetch categories:", error);
    }
  };

  const fetchParts = async () => {
    setLoading(true);
    try {
      const params: any = {
        page: currentPage,
        limit: pageSize,
      };
      
      if (searchQuery.trim()) {
        params.q = searchQuery.trim();
      }
      
      if (selectedCategory !== "all") {
        params.category = selectedCategory;
      }

      const response: ApiResponse<Part[]> = await api.getParts(params);
      setParts(response.data || []);
      setTotalParts(response.meta?.total || 0);
    } catch (error) {
      toast.error("Failed to fetch parts"); console.error("Failed to fetch parts:", error);
      setParts([]);
      setTotalParts(0);
    } finally {
      setLoading(false);
    }
  };

  const handleRowClick = (ipn: string) => {
    navigate(`/parts/${encodeURIComponent(ipn)}`);
  };

  const handleSearch = (value: string) => {
    setSearchQuery(value);
    setCurrentPage(1); // Reset to first page on search
  };

  const handleCategoryChange = (value: string) => {
    setSelectedCategory(value);
    setCurrentPage(1); // Reset to first page on filter change
  };

  const handleReset = () => {
    setSearchQuery("");
    setSelectedCategory("all");
    setCurrentPage(1);
  };

  const selectedCategoryColumns = useMemo(() => {
    if (!partForm.category) return [];
    const cat = categories.find(c => c.id === partForm.category);
    return cat?.columns?.filter(c => c.toLowerCase() !== "ipn") || [];
  }, [partForm.category, categories]);

  const handleCreatePart = async () => {
    setCreating(true);
    setCreateError("");
    setIpnError("");
    try {
      // Check for duplicate IPN
      const check = await api.checkIPN(partForm.ipn);
      if (check.exists) {
        setIpnError("This IPN already exists");
        setCreating(false);
        return;
      }

      await api.createPart({
        ipn: partForm.ipn,
        category: partForm.category,
        fields: partForm.dynamicFields,
      });
      setCreateDialogOpen(false);
      setPartForm({ ipn: "", category: "", dynamicFields: {} });
      fetchParts();
    } catch (error: any) {
      const msg = error?.message || "Failed to create part";
      if (msg.includes("already exists")) {
        setIpnError("This IPN already exists");
      } else {
        setCreateError(msg);
      }
    } finally {
      setCreating(false);
    }
  };

  const handleCreateCategory = async () => {
    setCreatingCategory(true);
    try {
      await api.createCategory(newCatData);
      setNewCatDialogOpen(false);
      setNewCatData({ title: "", prefix: "" });
      await fetchCategories();
    } catch (error: any) {
      toast.error("Failed to create category"); console.error("Failed to create category:", error);
    } finally {
      setCreatingCategory(false);
    }
  };

  // Calculate pagination
  const totalPages = Math.ceil(totalParts / pageSize);
  const hasNextPage = currentPage < totalPages;
  const hasPrevPage = currentPage > 1;

  // Extract fields for display
  const displayParts = useMemo(() => {
    return parts.map(part => {
      const fields = part.fields || {};
      return {
        ...part,
        category: fields._category || fields.category || 'Unknown',
        description: fields.description || fields.desc || '',
        cost: parseFloat(fields.cost || fields.unit_price || '0') || undefined,
        stock: parseFloat(fields.stock || fields.qty_on_hand || fields.current_stock || '0') || undefined,
        status: fields.status || 'active',
      } as PartWithFields;
    });
  }, [parts]);

  const partsColumns: ColumnDef<PartWithFields>[] = [
    {
      id: "ipn",
      label: "IPN",
      accessor: (part) => <span className="font-mono font-medium">{part.ipn}</span>,
      sortValue: (part) => part.ipn,
      defaultVisible: true,
    },
    {
      id: "category",
      label: "Category",
      accessor: (part) => <Badge variant="secondary" className="capitalize">{part.category}</Badge>,
      sortValue: (part) => part.category || "",
      defaultVisible: true,
    },
    {
      id: "description",
      label: "Description",
      accessor: (part) => <span className="max-w-xs truncate block">{part.description || "No description"}</span>,
      sortValue: (part) => part.description || "",
      defaultVisible: true,
    },
    {
      id: "cost",
      label: "Cost",
      accessor: (part) => part.cost ? `$${part.cost.toFixed(2)}` : "—",
      sortValue: (part) => part.cost || 0,
      className: "text-right",
      headerClassName: "text-right",
      defaultVisible: true,
    },
    {
      id: "stock",
      label: "Stock",
      accessor: (part) => part.stock !== undefined ? part.stock.toString() : "—",
      sortValue: (part) => part.stock ?? 0,
      className: "text-right",
      headerClassName: "text-right",
      defaultVisible: true,
    },
    {
      id: "status",
      label: "Status",
      accessor: (part) => (
        <Badge variant={part.status === "active" ? "default" : "secondary"}>
          {part.status || "active"}
        </Badge>
      ),
      sortValue: (part) => part.status || "active",
      defaultVisible: true,
    },
    ...(gitplmConfigured ? [{
      id: "gitplm" as const,
      label: "GitPLM",
      accessor: (part: PartWithFields) => {
        const url = gitplmUrl(part.ipn);
        return url ? (
          <a
            href={url}
            target="_blank"
            rel="noopener noreferrer"
            onClick={(e) => e.stopPropagation()}
            className="text-muted-foreground hover:text-primary"
            title="Open in gitplm"
          >
            <ExternalLink className="h-4 w-4" />
          </a>
        ) : null;
      },
      defaultVisible: true,
    }] : []),
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Parts</h1>
          <p className="text-muted-foreground">
            Manage your parts inventory and specifications
          </p>
        </div>
        <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Add Part
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-[600px]">
            <DialogHeader>
              <DialogTitle>Add New Part</DialogTitle>
              <DialogDescription>
                Create a new part in your inventory system.
              </DialogDescription>
            </DialogHeader>
            
            {createError && (
              <div className="text-sm text-destructive bg-destructive/10 p-2 rounded" data-testid="create-error">
                {createError}
              </div>
            )}

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="ipn">IPN *</Label>
                <Input
                  id="ipn"
                  placeholder="Internal Part Number"
                  value={partForm.ipn}
                  onChange={(e) => {
                    setPartForm(prev => ({ ...prev, ipn: e.target.value }));
                    setIpnError("");
                  }}
                />
                {ipnError && <p className="text-sm text-destructive" data-testid="ipn-error">{ipnError}</p>}
              </div>
              
              <div className="space-y-2">
                <Label>Category *</Label>
                <div className="flex gap-2">
                  <Select
                    value={partForm.category}
                    onValueChange={(value) => setPartForm(prev => ({ ...prev, category: value, dynamicFields: {} }))}
                  >
                    <SelectTrigger className="flex-1">
                      <SelectValue placeholder="Select category" />
                    </SelectTrigger>
                    <SelectContent>
                      {categories.map((category) => (
                        <SelectItem key={category.id} value={category.id}>
                          {category.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <Button type="button" variant="outline" size="icon" onClick={() => setNewCatDialogOpen(true)} title="New Category">
                    <Plus className="h-4 w-4" />
                  </Button>
                </div>
              </div>

              {selectedCategoryColumns.map((col) => (
                <div key={col} className="space-y-2">
                  <Label htmlFor={`field-${col}`} className="capitalize">{col}</Label>
                  <Input
                    id={`field-${col}`}
                    placeholder={col}
                    value={partForm.dynamicFields[col] || ""}
                    onChange={(e) => setPartForm(prev => ({
                      ...prev,
                      dynamicFields: { ...prev.dynamicFields, [col]: e.target.value }
                    }))}
                  />
                </div>
              ))}
            </div>
            
            <DialogFooter>
              <Button 
                type="button" 
                variant="outline" 
                onClick={() => setCreateDialogOpen(false)}
                disabled={creating}
              >
                Cancel
              </Button>
              <Button 
                onClick={handleCreatePart}
                disabled={creating || !partForm.ipn || !partForm.category}
              >
                {creating ? 'Creating...' : 'Create Part'}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* New Category Dialog */}
        <Dialog open={newCatDialogOpen} onOpenChange={setNewCatDialogOpen}>
          <DialogContent className="sm:max-w-[400px]">
            <DialogHeader>
              <DialogTitle>Create New Category</DialogTitle>
              <DialogDescription>
                Add a new category for organizing parts.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="cat-title">Title</Label>
                <Input
                  id="cat-title"
                  placeholder="e.g., Connectors"
                  value={newCatData.title}
                  onChange={(e) => setNewCatData(prev => ({ ...prev, title: e.target.value }))}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="cat-prefix">Prefix</Label>
                <Input
                  id="cat-prefix"
                  placeholder="e.g., CON"
                  value={newCatData.prefix}
                  onChange={(e) => setNewCatData(prev => ({ ...prev, prefix: e.target.value.toUpperCase() }))}
                />
                <p className="text-xs text-muted-foreground">
                  Will create category file: z-{newCatData.prefix.toLowerCase() || "xxx"}.csv
                </p>
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setNewCatDialogOpen(false)} disabled={creatingCategory}>
                Cancel
              </Button>
              <Button onClick={handleCreateCategory} disabled={creatingCategory || !newCatData.title || !newCatData.prefix}>
                {creatingCategory ? "Creating..." : "Create Category"}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      {/* Filters Card */}
      <Card>
        <CardHeader className="pb-4">
          <CardTitle className="text-base font-medium">Filters</CardTitle>
        </CardHeader>
        <CardContent>
          {showScanner && (
            <div className="mb-4">
              <BarcodeScanner
                onScan={(code) => {
                  handleSearch(code);
                  setShowScanner(false);
                }}
              />
            </div>
          )}
          <div className="flex flex-col sm:flex-row gap-4">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  placeholder="Search parts by IPN, description..."
                  value={searchQuery}
                  onChange={(e) => handleSearch(e.target.value)}
                  className="pl-10"
                />
              </div>
            </div>
            <Button
              variant="outline"
              onClick={() => setShowScanner(!showScanner)}
            >
              <ScanLine className="h-4 w-4 mr-1" />
              Scan
            </Button>
            <div className="w-full sm:w-48">
              <Select value={selectedCategory} onValueChange={handleCategoryChange}>
                <SelectTrigger>
                  <Filter className="h-4 w-4 mr-2" />
                  <SelectValue placeholder="Category" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Categories</SelectItem>
                  {categories.map((category) => (
                    <SelectItem key={category.id} value={category.id}>
                      {category.name} ({category.count})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <Button variant="outline" onClick={handleReset}>
              <RotateCcw className="h-4 w-4" />
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Results */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>
            Parts ({totalParts.toLocaleString()})
          </CardTitle>
          <div className="text-sm text-muted-foreground">
            Page {currentPage} of {totalPages}
          </div>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="space-y-3">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : (
            <>
              <ConfigurableTable<PartWithFields>
                tableName="parts"
                columns={partsColumns}
                data={displayParts}
                rowKey={(part) => part.ipn}
                onRowClick={(part) => handleRowClick(part.ipn)}
                emptyMessage={
                  searchQuery || selectedCategory !== "all"
                    ? "No parts found matching your criteria"
                    : "No parts available"
                }
              />

              {/* Pagination */}
              {totalPages > 1 && (
                <div className="flex items-center justify-between mt-6">
                  <div className="text-sm text-muted-foreground">
                    Showing {((currentPage - 1) * pageSize) + 1} to {Math.min(currentPage * pageSize, totalParts)} of {totalParts} parts
                  </div>
                  <div className="flex items-center space-x-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setCurrentPage(currentPage - 1)}
                      disabled={!hasPrevPage}
                    >
                      <ChevronLeft className="h-4 w-4 mr-1" />
                      Previous
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setCurrentPage(currentPage + 1)}
                      disabled={!hasNextPage}
                    >
                      Next
                      <ChevronRight className="h-4 w-4 ml-1" />
                    </Button>
                  </div>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
export default Parts;
