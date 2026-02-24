import { useEffect, useState, useMemo, lazy, Suspense, useRef } from "react";
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "../components/ui/dropdown-menu";
// Table components used by ConfigurableTable internally
import { Skeleton } from "../components/ui/skeleton";
import { LoadingState } from "../components/LoadingState";
import { ErrorState } from "../components/ErrorState";
import { 
  Search, 
  Filter,
  ScanLine,
  ChevronLeft,
  ChevronRight,
  RotateCcw,
  Plus,
  Download,
  Package
} from "lucide-react";
import { api, type Part, type Category, type ApiResponse } from "../lib/api";
import { ConfigurableTable, type ColumnDef } from "../components/ConfigurableTable";
// Lazy load BarcodeScanner to reduce initial bundle size (329KB chunk)
const BarcodeScanner = lazy(() => import("../components/BarcodeScanner").then(m => ({ default: m.BarcodeScanner })));
import { useGitPLM } from "../hooks/useGitPLM";
import { ExternalLink } from "lucide-react";
import { toast } from "sonner";
import { useHotkeys } from 'react-hotkeys-hook';
import { KeyboardShortcutsHelp } from '../components/KeyboardShortcutsHelp';
import { useForm } from "react-hook-form";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "../components/ui/form";
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
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [showScanner, setShowScanner] = useState(false);
  const [selectedCategory, setSelectedCategory] = useState<string>("all");
  const [currentPage, setCurrentPage] = useState(1);
  const [totalParts, setTotalParts] = useState(0);
  const { configured: gitplmConfigured, buildUrl: gitplmUrl } = useGitPLM();
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [newCatDialogOpen, setNewCatDialogOpen] = useState(false);
  const [newCatData, setNewCatData] = useState<NewCategoryData>({ title: "", prefix: "" });
  const [creatingCategory, setCreatingCategory] = useState(false);
  const pageSize = 50;
  
  const searchInputRef = useRef<HTMLInputElement>(null);

  const partForm = useForm<CreatePartData>({
    defaultValues: {
      ipn: "",
      category: "",
      dynamicFields: {},
    },
  });

  // Keyboard shortcuts
  useHotkeys('n', () => setCreateDialogOpen(true), { 
    enableOnFormTags: false,
    preventDefault: true 
  });
  
  useHotkeys('/', () => {
    searchInputRef.current?.focus();
  }, { 
    enableOnFormTags: false,
    preventDefault: true 
  });

  useHotkeys('escape', () => {
    if (createDialogOpen) setCreateDialogOpen(false);
    if (newCatDialogOpen) setNewCatDialogOpen(false);
  }, {
    enableOnFormTags: true,
    preventDefault: true
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
    } catch (error: any) {
      const errorMessage = error?.message || "Network error";
      toast.error(`Failed to fetch categories: ${errorMessage}`);
      console.error("Failed to fetch categories:", error);
    }
  };

  const fetchParts = async () => {
    setLoading(true);
    setError(null);
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
    } catch (error: any) {
      const errorMessage = error?.message || "Network error";
      toast.error(`Failed to fetch parts: ${errorMessage}`);
      console.error("Failed to fetch parts:", error);
      setParts([]);
      setTotalParts(0);
      setError(errorMessage);
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

  const handleExport = (format: 'csv' | 'xlsx') => {
    const params = new URLSearchParams();
    params.set('format', format);
    if (searchQuery.trim()) {
      params.set('search', searchQuery.trim());
    }
    if (selectedCategory !== 'all') {
      params.set('category', selectedCategory);
    }
    window.location.href = `/api/v1/parts/export?${params.toString()}`;
    toast.success(`Exporting parts as ${format.toUpperCase()}`);
  };

  const handleReset = () => {
    setSearchQuery("");
    setSelectedCategory("all");
    setCurrentPage(1);
  };

  const selectedPartCategory = partForm.watch("category");
  const selectedCategoryColumns = useMemo(() => {
    if (!selectedPartCategory) return [];
    const cat = categories.find(c => c.id === selectedPartCategory);
    return cat?.columns?.filter(c => c.toLowerCase() !== "ipn") || [];
  }, [selectedPartCategory, categories]);

  const handleCreatePart = async (data: CreatePartData) => {
    setCreating(true);
    try {
      // Check for duplicate IPN
      const check = await api.checkIPN(data.ipn);
      if (check.exists) {
        partForm.setError("ipn", { message: "This IPN already exists" });
        setCreating(false);
        return;
      }

      await api.createPart({
        ipn: data.ipn,
        category: data.category,
        fields: data.dynamicFields,
      });
      toast.success(`Part ${data.ipn} created successfully`);
      setCreateDialogOpen(false);
      partForm.reset();
      fetchParts();
    } catch (error: any) {
      const msg = error?.message || "Failed to create part";
      if (msg.toLowerCase().includes("already exists") || msg.toLowerCase().includes("duplicate")) {
        partForm.setError("ipn", { message: "This IPN already exists" });
      } else {
        toast.error(`Failed to create part: ${msg}`);
      }
    } finally {
      setCreating(false);
    }
  };

  const handleCreateCategory = async () => {
    setCreatingCategory(true);
    try {
      await api.createCategory(newCatData);
      toast.success(`Category "${newCatData.title}" created successfully`);
      setNewCatDialogOpen(false);
      setNewCatData({ title: "", prefix: "" });
      await fetchCategories();
    } catch (error: any) {
      const errorMessage = error?.message || "Failed to create category";
      toast.error(`Failed to create category: ${errorMessage}`);
      console.error("Failed to create category:", error);
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
      id: "manufacturer",
      label: "Manufacturer",
      accessor: (part) => {
        const mfg = part.fields?.manufacturer || part.fields?.mfg;
        return mfg ? <span className="text-sm">{mfg}</span> : <span className="text-muted-foreground">—</span>;
      },
      sortValue: (part) => part.fields?.manufacturer || part.fields?.mfg || "",
      defaultVisible: false, // Hidden by default but available
    },
    {
      id: "mpn",
      label: "MPN",
      accessor: (part) => {
        const mpn = part.fields?.mpn || part.fields?.manufacturer_part_number;
        return mpn ? <span className="font-mono text-sm">{mpn}</span> : <span className="text-muted-foreground">—</span>;
      },
      sortValue: (part) => part.fields?.mpn || part.fields?.manufacturer_part_number || "",
      defaultVisible: false, // Hidden by default but available
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
      <div className="flex flex-col sm:flex-row items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl sm:text-3xl font-bold tracking-tight">Parts</h1>
          <p className="text-muted-foreground text-sm sm:text-base">
            Manage your parts inventory and specifications
          </p>
        </div>
        <div className="flex gap-2 w-full sm:w-auto">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" className="min-h-[44px] flex-1 sm:flex-none" aria-label="Export parts data">
                <Download className="h-4 w-4 sm:mr-2" aria-hidden="true" />
                <span className="hidden sm:inline" aria-hidden="true">Export</span>
                <span className="sr-only">Export parts data</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              <DropdownMenuItem onClick={() => handleExport('csv')}>
                Export as CSV
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => handleExport('xlsx')}>
                Export as Excel
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
            <DialogTrigger asChild>
              <Button className="min-h-[44px] flex-1 sm:flex-none">
                <Plus className="h-4 w-4 sm:mr-2" />
                <span className="hidden sm:inline">Add Part</span>
                <span className="sm:hidden">Add</span>
              </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-[600px]">
            <Form {...partForm}>
              <form onSubmit={partForm.handleSubmit(handleCreatePart)} className="space-y-6">
                <DialogHeader>
                  <DialogTitle>Add New Part</DialogTitle>
                  <DialogDescription>
                    Create a new part in your inventory system.
                  </DialogDescription>
                </DialogHeader>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <FormField
                    control={partForm.control}
                    name="ipn"
                    rules={{ required: 'IPN is required' }}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>IPN *</FormLabel>
                        <FormControl>
                          <Input placeholder="Internal Part Number" {...field} />
                        </FormControl>
                        <FormMessage data-testid="ipn-error" />
                      </FormItem>
                    )}
                  />
                  
                  <FormField
                    control={partForm.control}
                    name="category"
                    rules={{ required: 'Category is required' }}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Category *</FormLabel>
                        <div className="flex gap-2">
                          <Select
                            value={field.value}
                            onValueChange={(value) => {
                              field.onChange(value);
                              partForm.setValue("dynamicFields", {});
                            }}
                          >
                            <FormControl>
                              <SelectTrigger className="flex-1">
                                <SelectValue placeholder="Select category" />
                              </SelectTrigger>
                            </FormControl>
                            <SelectContent>
                              {categories.map((category) => (
                                <SelectItem key={category.id} value={category.id}>
                                  {category.name}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                          <Button type="button" variant="outline" size="icon" onClick={() => setNewCatDialogOpen(true)} aria-label="Add new category" title="Add new category">
                            <Plus className="h-4 w-4" aria-hidden="true" />
                          </Button>
                        </div>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  {selectedCategoryColumns.map((col) => (
                    <FormField
                      key={col}
                      control={partForm.control}
                      name={`dynamicFields.${col}` as any}
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel className="capitalize">{col}</FormLabel>
                          <FormControl>
                            <Input placeholder={col} {...field} />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
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
                    type="submit"
                    disabled={creating}
                  >
                    {creating ? 'Creating...' : 'Create Part'}
                  </Button>
                </DialogFooter>
              </form>
            </Form>
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
      </div>

      {/* Filters Card */}
      <Card>
        <CardHeader className="pb-4">
          <CardTitle className="text-base font-medium">Filters</CardTitle>
        </CardHeader>
        <CardContent>
          {showScanner && (
            <div className="mb-4">
              <Suspense fallback={<Skeleton className="h-64 w-full" />}>
                <BarcodeScanner
                  onScan={(code) => {
                    handleSearch(code);
                    setShowScanner(false);
                  }}
                />
              </Suspense>
            </div>
          )}
          <div className="flex flex-col sm:flex-row gap-4">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  ref={searchInputRef}
                  placeholder="Search parts by IPN, description... (Press / to focus)"
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
            <Button variant="outline" onClick={handleReset} aria-label="Reset filters">
              <RotateCcw className="h-4 w-4" aria-hidden="true" />
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
            <LoadingState variant="table" rows={5} />
          ) : error ? (
            <ErrorState 
              title="Failed to load parts"
              message={error}
              onRetry={fetchParts}
            />
          ) : (
            <>
              <ConfigurableTable<PartWithFields>
                tableName="parts"
                columns={partsColumns}
                data={displayParts}
                rowKey={(part) => part.ipn}
                onRowClick={(part) => handleRowClick(part.ipn)}
                ariaLabel="Parts inventory list"
                caption="List of all parts with IPN, category, description, cost, and stock levels"
                emptyMessage={
                  searchQuery || selectedCategory !== "all"
                    ? "No parts found matching your criteria"
                    : "No parts available"
                }
                emptyIcon={Package}
                emptyDescription={
                  searchQuery || selectedCategory !== "all"
                    ? "Try adjusting your search or filters."
                    : "Get started by adding your first part."
                }
                emptyAction={
                  searchQuery || selectedCategory !== "all" ? undefined : (
                    <Button onClick={() => setCreateDialogOpen(true)}>
                      <Plus className="h-4 w-4 mr-2" />
                      Add Part
                    </Button>
                  )
                }
              />

              {/* Pagination */}
              {totalPages > 1 && (
                <div className="flex flex-col sm:flex-row items-center justify-between gap-4 mt-6">
                  <div className="text-sm text-muted-foreground text-center sm:text-left">
                    Showing {((currentPage - 1) * pageSize) + 1} to {Math.min(currentPage * pageSize, totalParts)} of {totalParts} parts
                  </div>
                  <div className="flex items-center space-x-2">
                    <Button
                      variant="outline"
                      size="default"
                      className="min-h-[44px]"
                      onClick={() => setCurrentPage(currentPage - 1)}
                      disabled={!hasPrevPage}
                    >
                      <ChevronLeft className="h-4 w-4 sm:mr-1" />
                      <span className="hidden sm:inline">Previous</span>
                    </Button>
                    <Button
                      variant="outline"
                      size="default"
                      className="min-h-[44px]"
                      onClick={() => setCurrentPage(currentPage + 1)}
                      disabled={!hasNextPage}
                    >
                      <span className="hidden sm:inline">Next</span>
                      <ChevronRight className="h-4 w-4 sm:ml-1" />
                    </Button>
                  </div>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>
      
      <KeyboardShortcutsHelp 
        shortcuts={[
          { key: 'n', description: 'Create new part' },
          { key: '/', description: 'Focus search input' },
          { key: 'Esc', description: 'Close dialogs' },
        ]}
      />
    </div>
  );
}
export default Parts;
