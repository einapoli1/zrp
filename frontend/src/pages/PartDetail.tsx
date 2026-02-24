import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Separator } from "../components/ui/separator";
import { Skeleton } from "../components/ui/skeleton";
import { 
  Package, 
  ChevronDown, 
  ChevronRight,
  DollarSign,
  Layers,
  Info,
  GitBranch,
  RefreshCw,
  Store,
  Plus,
  Check,
  Factory
} from "lucide-react";
import { Link } from "react-router-dom";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../components/ui/table";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { Checkbox } from "../components/ui/checkbox";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "../components/ui/dialog";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "../components/ui/alert-dialog";
import { Label } from "../components/ui/label";
import { api, type Part, type BOMNode, type PartCost, type WhereUsedEntry, type MarketPricingResult, type PartChange, type PartManufacturer } from "../lib/api";
import { useGitPLM } from "../hooks/useGitPLM";
import { ExternalLink, Edit, Trash2, FilePlus } from "lucide-react";
import { toast } from "sonner";
import { Breadcrumb } from "../components/ui/breadcrumb";
import { LoadingState } from "../components/LoadingState";
import { BOMEditor } from "../components/BOMEditor";

interface PartWithDetails extends Part {
  category?: string;
  description?: string;
  manufacturer?: string;
  mpn?: string;
  cost?: number;
  price?: number;
  stock?: number;
  location?: string;
  status?: string;
  datasheet?: string;
  notes?: string;
}

interface BOMTreeProps {
  node: BOMNode;
  level?: number;
  onPartClick?: (ipn: string) => void;
  gitplmBuildUrl?: (ipn: string) => string | null;
}

function BOMTree({ node, level = 0, onPartClick, gitplmBuildUrl }: BOMTreeProps) {
  const [expanded, setExpanded] = useState(level < 2); // Auto-expand first 2 levels
  const hasChildren = node.children && node.children.length > 0;
  
  const handleToggle = () => {
    if (hasChildren) {
      setExpanded(!expanded);
    }
  };

  const handlePartClick = (ipn: string) => {
    if (onPartClick) {
      onPartClick(ipn);
    }
  };

  return (
    <div className="select-none">
      <div 
        className={`flex items-center py-2 px-3 rounded-md hover:bg-muted/50 cursor-pointer ${
          level > 0 ? 'ml-' + (level * 4) : ''
        }`}
        onClick={handleToggle}
      >
        <div className="flex items-center min-w-0 flex-1">
          {hasChildren ? (
            expanded ? 
              <ChevronDown className="h-4 w-4 text-muted-foreground mr-2 flex-shrink-0" /> :
              <ChevronRight className="h-4 w-4 text-muted-foreground mr-2 flex-shrink-0" />
          ) : (
            <div className="w-6 mr-2 flex-shrink-0" />
          )}
          
          <div className="flex items-center min-w-0 flex-1">
            <span 
              className="font-mono text-sm font-medium text-primary hover:underline mr-1"
              onClick={(e) => {
                e.stopPropagation();
                handlePartClick(node.ipn);
              }}
            >
              {node.ipn}
            </span>
            {gitplmBuildUrl && gitplmBuildUrl(node.ipn) && (
              <a
                href={gitplmBuildUrl(node.ipn)!}
                target="_blank"
                rel="noopener noreferrer"
                onClick={(e) => e.stopPropagation()}
                className="text-muted-foreground hover:text-primary mr-2"
                title="Open in gitplm"
              >
                <ExternalLink className="h-3 w-3" />
              </a>
            )}
            <span className="text-sm text-muted-foreground truncate">
              {node.description || 'No description'}
            </span>
          </div>
          
          <div className="flex items-center space-x-3 ml-4">
            {node.qty && node.qty > 0 && (
              <Badge variant="outline" className="text-xs">
                Qty: {node.qty}
              </Badge>
            )}
            {node.ref && (
              <Badge variant="secondary" className="text-xs">
                {node.ref}
              </Badge>
            )}
          </div>
        </div>
      </div>
      
      {expanded && hasChildren && (
        <div className="ml-4 border-l-2 border-muted pl-2">
          {node.children.map((child, index) => (
            <BOMTree 
              key={`${child.ipn}-${index}`} 
              node={child} 
              level={level + 1}
              onPartClick={onPartClick}
              gitplmBuildUrl={gitplmBuildUrl}
            />
          ))}
        </div>
      )}
    </div>
  );
}

// Helper to flatten BOM tree to editable items (only direct children)
function flattenBOMToItems(bom: BOMNode): Array<{ id: string; ipn: string; description: string; quantity: number; ref_des: string }> {
  if (!bom.children || bom.children.length === 0) {
    return [];
  }
  
  return bom.children.map(child => ({
    id: crypto.randomUUID(),
    ipn: child.ipn,
    description: child.description || "",
    quantity: child.qty || 1,
    ref_des: child.ref || "",
  }));
}

function PartDetail() {
  const { ipn } = useParams<{ ipn: string }>();
  const navigate = useNavigate();
  const [part, setPart] = useState<PartWithDetails | null>(null);
  const [bom, setBom] = useState<BOMNode | null>(null);
  const [cost, setCost] = useState<PartCost | null>(null);
  const [loading, setLoading] = useState(true);
  const [bomLoading, setBomLoading] = useState(false);
  const [costLoading, setCostLoading] = useState(false);
  const [whereUsed, setWhereUsed] = useState<WhereUsedEntry[]>([]);
  const [whereUsedLoading, setWhereUsedLoading] = useState(false);
  const [marketPricing, setMarketPricing] = useState<MarketPricingResult[]>([]);
  const [marketPricingLoading, setMarketPricingLoading] = useState(false);
  const [marketPricingCached, setMarketPricingCached] = useState(false);
  const [marketPricingError, setMarketPricingError] = useState<string>("");
  const [marketPricingUnconfigured, setMarketPricingUnconfigured] = useState<string[]>([]);
  const [editing, setEditing] = useState(false);
  const [editFields, setEditFields] = useState<Record<string, string>>({});
  const [pendingChanges, setPendingChanges] = useState<PartChange[]>([]);
  const [savingChanges, setSavingChanges] = useState(false);
  const [editingBOM, setEditingBOM] = useState(false);
  const [, setPendingLoading] = useState(false);
  const [creatingECO, setCreatingECO] = useState(false);
  const { configured: gitplmConfigured, buildUrl: gitplmUrl } = useGitPLM();
  
  // Manufacturers state (normalized architecture)
  const [manufacturers, setManufacturers] = useState<PartManufacturer[]>([]);
  const [manufacturersLoading, setManufacturersLoading] = useState(false);
  const [manufacturerDialogOpen, setManufacturerDialogOpen] = useState(false);
  const [editingManufacturer, setEditingManufacturer] = useState<PartManufacturer | null>(null);
  const [manufacturerForm, setManufacturerForm] = useState({
    manufacturer_id: 0,
    manufacturer_name: "",
    mpn: "",
    is_primary: false,
    approved: true,
    notes: ""
  });
  const [manufacturerFormErrors, setManufacturerFormErrors] = useState<Record<string, string>>({});
  const [savingManufacturer, setSavingManufacturer] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [manufacturerToDelete, setManufacturerToDelete] = useState<PartManufacturer | null>(null);
  const [deletingManufacturer, setDeletingManufacturer] = useState(false);
  
  // Manufacturer autocomplete state
  const [availableManufacturers, setAvailableManufacturers] = useState<Array<{ id: number; name: string; contact_email?: string }>>([]);
  const [manufacturerSearch, setManufacturerSearch] = useState("");
  const [showManufacturerDropdown, setShowManufacturerDropdown] = useState(false);

  const fetchPendingChanges = async () => {
    if (!ipn) return;
    setPendingLoading(true);
    try {
      const data = await api.getPartChanges(decodeURIComponent(ipn));
      setPendingChanges(data.filter((c: PartChange) => c.status === 'draft' || c.status === 'pending'));
    } catch {
      // ignore
    } finally {
      setPendingLoading(false);
    }
  };

  const handleStartEdit = () => {
    if (!part) return;
    // Initialize edit fields from current part fields
    const fields: Record<string, string> = {};
    const excluded = ['_category'];
    for (const [k, v] of Object.entries(part.fields || {})) {
      if (!excluded.includes(k)) {
        fields[k] = v;
      }
    }
    setEditFields(fields);
    setEditing(true);
  };

  const handleSaveChanges = async () => {
    if (!ipn || !part) return;
    setSavingChanges(true);
    try {
      const changes: Array<{ field_name: string; old_value: string; new_value: string }> = [];
      const currentFields = part.fields || {};
      for (const [k, v] of Object.entries(editFields)) {
        const oldVal = currentFields[k] || '';
        if (v !== oldVal) {
          changes.push({ field_name: k, old_value: oldVal, new_value: v });
        }
      }
      if (changes.length === 0) {
        setEditing(false);
        return;
      }
      await api.createPartChanges(decodeURIComponent(ipn), changes);
      setEditing(false);
      fetchPendingChanges();
    } catch (error) {
      toast.error("Failed to save changes"); console.error("Failed to save changes:", error);
    } finally {
      setSavingChanges(false);
    }
  };

  const handleDeleteChange = async (changeId: number) => {
    if (!ipn) return;
    try {
      await api.deletePartChange(decodeURIComponent(ipn), changeId);
      fetchPendingChanges();
    } catch (error) {
      toast.error("Failed to delete change"); console.error("Failed to delete change:", error);
    }
  };

  const handleCreateECO = async () => {
    if (!ipn) return;
    setCreatingECO(true);
    try {
      const result = await api.createECOFromPartChanges(decodeURIComponent(ipn), {
        title: `Part changes for ${ipn}`,
      });
      navigate(`/ecos/${result.eco_id}`);
    } catch (error) {
      toast.error("Failed to create ECO"); console.error("Failed to create ECO:", error);
    } finally {
      setCreatingECO(false);
    }
  };

  // Manufacturers handlers (normalized with autocomplete)
  const fetchManufacturers = async () => {
    if (!ipn) return;
    setManufacturersLoading(true);
    try {
      const data = await api.getPartManufacturers(decodeURIComponent(ipn));
      setManufacturers(data.manufacturers || []);
    } catch (error) {
      console.error("Failed to fetch manufacturers:", error);
      // Don't show toast - it's ok if manufacturers don't exist yet
    } finally {
      setManufacturersLoading(false);
    }
  };

  const fetchAvailableManufacturers = async () => {
    try {
      const data = await api.getManufacturers({ approved: true });
      setAvailableManufacturers(data);
    } catch (error) {
      console.error("Failed to fetch available manufacturers:", error);
    }
  };

  const openAddManufacturerDialog = () => {
    setEditingManufacturer(null);
    setManufacturerForm({
      manufacturer_id: 0,
      manufacturer_name: "",
      mpn: "",
      is_primary: manufacturers.length === 0, // Default to primary if no manufacturers exist
      approved: true,
      notes: ""
    });
    setManufacturerSearch("");
    setManufacturerFormErrors({});
    setManufacturerDialogOpen(true);
    fetchAvailableManufacturers();
  };

  const openEditManufacturerDialog = (mfg: PartManufacturer) => {
    setEditingManufacturer(mfg);
    setManufacturerForm({
      manufacturer_id: mfg.manufacturer_id,
      manufacturer_name: mfg.manufacturer_name,
      mpn: mfg.mpn,
      is_primary: mfg.is_primary,
      approved: mfg.approved,
      notes: mfg.notes || ""
    });
    setManufacturerSearch(mfg.manufacturer_name);
    setManufacturerFormErrors({});
    setManufacturerDialogOpen(true);
    fetchAvailableManufacturers();
  };

  const validateManufacturerForm = (): boolean => {
    const errors: Record<string, string> = {};
    
    if (!manufacturerForm.manufacturer_id || manufacturerForm.manufacturer_id === 0) {
      errors.manufacturer = "Manufacturer is required";
    }
    if (!manufacturerForm.mpn.trim()) {
      errors.mpn = "MPN is required";
    }
    
    // Warn if unchecking primary and no other manufacturer is primary
    if (!manufacturerForm.is_primary && editingManufacturer?.is_primary) {
      const otherPrimary = manufacturers.find(m => m.id !== editingManufacturer.id && m.is_primary);
      if (!otherPrimary) {
        errors.is_primary = "At least one manufacturer must be primary";
      }
    }
    
    setManufacturerFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleManufacturerSelect = (mfg: { id: number; name: string; contact_email?: string }) => {
    setManufacturerForm({
      ...manufacturerForm,
      manufacturer_id: mfg.id,
      manufacturer_name: mfg.name,
    });
    setManufacturerSearch(mfg.name);
    setShowManufacturerDropdown(false);
  };

  const filteredManufacturers = availableManufacturers.filter(
    (m) =>
      manufacturerSearch === "" ||
      m.name.toLowerCase().includes(manufacturerSearch.toLowerCase()) ||
      m.contact_email?.toLowerCase().includes(manufacturerSearch.toLowerCase())
  );

  const handleSaveManufacturer = async () => {
    if (!ipn) return;
    
    if (!validateManufacturerForm()) {
      return;
    }
    
    setSavingManufacturer(true);
    try {
      if (editingManufacturer) {
        // Update existing
        await api.updatePartManufacturer(decodeURIComponent(ipn), editingManufacturer.id, {
          manufacturer_id: manufacturerForm.manufacturer_id,
          mpn: manufacturerForm.mpn,
          is_primary: manufacturerForm.is_primary,
          approved: manufacturerForm.approved,
          notes: manufacturerForm.notes,
        });
        toast.success("Manufacturer updated successfully");
      } else {
        // Create new
        await api.createPartManufacturer(decodeURIComponent(ipn), {
          manufacturer_id: manufacturerForm.manufacturer_id,
          mpn: manufacturerForm.mpn,
          is_primary: manufacturerForm.is_primary,
          approved: manufacturerForm.approved,
          notes: manufacturerForm.notes,
        });
        toast.success("Manufacturer added successfully");
      }
      
      setManufacturerDialogOpen(false);
      fetchManufacturers();
    } catch (error: any) {
      const errorMessage = error?.message || "Failed to save manufacturer";
      toast.error(errorMessage);
      console.error("Failed to save manufacturer:", error);
    } finally {
      setSavingManufacturer(false);
    }
  };

  const openDeleteManufacturerDialog = (mfg: PartManufacturer) => {
    setManufacturerToDelete(mfg);
    setDeleteDialogOpen(true);
  };

  const handleDeleteManufacturer = async () => {
    if (!ipn || !manufacturerToDelete) return;
    
    setDeletingManufacturer(true);
    try {
      await api.deletePartManufacturer(decodeURIComponent(ipn), manufacturerToDelete.id);
      toast.success("Manufacturer deleted successfully");
      setDeleteDialogOpen(false);
      setManufacturerToDelete(null);
      fetchManufacturers();
    } catch (error: any) {
      const errorMessage = error?.message || "Failed to delete manufacturer";
      toast.error(errorMessage);
      console.error("Failed to delete manufacturer:", error);
    } finally {
      setDeletingManufacturer(false);
    }
  };

  useEffect(() => {
    if (ipn) {
      fetchPartDetails();
      fetchPendingChanges();
      fetchManufacturers();
    }
  }, [ipn]);

  const fetchPartDetails = async () => {
    if (!ipn) return;
    
    setLoading(true);
    try {
      const partData = await api.getPart(decodeURIComponent(ipn));
      
      // Transform fields for display
      const fields = partData.fields || {};
      const detailedPart: PartWithDetails = {
        ...partData,
        category: fields._category || fields.category,
        description: fields.description || fields.desc,
        manufacturer: fields.manufacturer || fields.mfg,
        mpn: fields.mpn || fields.manufacturer_part_number,
        cost: parseFloat(fields.cost || fields.unit_cost || '0') || undefined,
        price: parseFloat(fields.price || fields.unit_price || '0') || undefined,
        stock: parseFloat(fields.stock || fields.qty_on_hand || fields.current_stock || '0') || undefined,
        location: fields.location,
        status: fields.status || 'active',
        datasheet: fields.datasheet || fields.datasheet_url,
        notes: fields.notes || fields.comments,
      };
      
      setPart(detailedPart);

      // Load BOM if this is an assembly
      const upperIPN = ipn.toUpperCase();
      if (upperIPN.startsWith('PCA-') || upperIPN.startsWith('ASY-')) {
        fetchBOM();
      }

      // Load cost information
      fetchCost();

      // Load where-used
      fetchWhereUsed();

      // Load market pricing if part has MPN
      if (detailedPart.mpn) {
        fetchMarketPricing(false);
      }
    } catch (error) {
      toast.error("Failed to fetch part details"); console.error("Failed to fetch part details:", error);
    } finally {
      setLoading(false);
    }
  };

  const fetchBOM = async () => {
    if (!ipn) return;
    
    setBomLoading(true);
    try {
      const bomData = await api.getPartBOM(decodeURIComponent(ipn));
      setBom(bomData);
    } catch (error) {
      toast.error("Failed to fetch BOM"); console.error("Failed to fetch BOM:", error);
    } finally {
      setBomLoading(false);
    }
  };

  const fetchCost = async () => {
    if (!ipn) return;
    
    setCostLoading(true);
    try {
      const costData = await api.getPartCost(decodeURIComponent(ipn));
      setCost(costData);
    } catch (error) {
      toast.error("Failed to fetch cost data"); console.error("Failed to fetch cost data:", error);
    } finally {
      setCostLoading(false);
    }
  };

  const fetchWhereUsed = async () => {
    if (!ipn) return;
    setWhereUsedLoading(true);
    try {
      const data = await api.getPartWhereUsed(decodeURIComponent(ipn));
      setWhereUsed(data);
    } catch (error) {
      toast.error("Failed to fetch where-used"); console.error("Failed to fetch where-used:", error);
    } finally {
      setWhereUsedLoading(false);
    }
  };

  const [marketNotConfigured, setMarketNotConfigured] = useState(false);

  const fetchMarketPricing = async (refresh: boolean) => {
    if (!ipn) return;
    setMarketPricingLoading(true);
    try {
      const data = await api.getMarketPricing(decodeURIComponent(ipn), refresh);
      if (data.not_configured) {
        setMarketNotConfigured(true);
        setMarketPricing([]);
        setMarketPricingError(data.error || "");
      } else {
        setMarketNotConfigured(false);
        setMarketPricing(data.results || []);
        setMarketPricingCached(data.cached || false);
        setMarketPricingError(data.error || "");
        setMarketPricingUnconfigured(data.unconfigured || []);
      }
      if (data.errors && data.errors.length > 0) {
        setMarketPricingError(prev => prev ? prev + "; " + data.errors!.join("; ") : data.errors!.join("; "));
      }
    } catch (error) {
      toast.error("Failed to fetch market pricing"); console.error("Failed to fetch market pricing:", error);
    } finally {
      setMarketPricingLoading(false);
    }
  };

  const handleBOMPartClick = (bomIPN: string) => {
    navigate(`/parts/${encodeURIComponent(bomIPN)}`);
  };

  const isAssembly = ipn && (ipn.toUpperCase().startsWith('PCA-') || ipn.toUpperCase().startsWith('ASY-'));

  if (loading) {
    return <LoadingState variant="spinner" message="Loading part..." />;
  }

  if (!part) {
    return (
      <div className="space-y-6">
        <Breadcrumb items={[
          { label: "Parts", href: "/parts" },
          { label: "Not Found" }
        ]} />
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Package className="h-12 w-12 text-muted-foreground mb-4" />
            <h3 className="text-lg font-semibold mb-2">Part Not Found</h3>
            <p className="text-muted-foreground text-center">
              The part with IPN "{ipn}" could not be found.
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Breadcrumb items={[
        { label: "Parts", href: "/parts" },
        { label: part.ipn }
      ]} />

      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight font-mono">{part.ipn}</h1>
          <p className="text-muted-foreground">
            {part.description || 'No description available'}
          </p>
        </div>
        <div className="flex items-center space-x-2">
          <Badge variant="secondary" className="capitalize">
            {part.category || 'Unknown'}
          </Badge>
          <Badge variant={part.status === 'active' ? 'default' : 'secondary'}>
            {part.status || 'active'}
          </Badge>
          <Button variant="default" size="sm" onClick={handleStartEdit} data-testid="edit-part-btn">
            <Edit className="h-4 w-4 mr-2" />
            Edit Part
          </Button>
          {gitplmConfigured && ipn && (
            <Button variant="outline" size="sm" asChild>
              <a href={gitplmUrl(ipn)!} target="_blank" rel="noopener noreferrer">
                <ExternalLink className="h-4 w-4 mr-2" />
                Edit in gitplm
              </a>
            </Button>
          )}
        </div>
      </div>

      {/* Edit Form */}
      {editing && (
        <Card data-testid="edit-form">
          <CardHeader>
            <CardTitle className="flex items-center">
              <Edit className="h-5 w-5 mr-2" />
              Edit Part Fields
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Changes are saved as pending drafts. You must create an ECO to apply them.
            </p>
            <div className="grid grid-cols-2 gap-4">
              {Object.entries(editFields).map(([key, value]) => (
                <div key={key}>
                  <label className="text-sm font-medium">{key}</label>
                  <Input
                    value={value}
                    onChange={(e) => setEditFields(prev => ({ ...prev, [key]: e.target.value }))}
                    data-testid={`edit-field-${key}`}
                  />
                </div>
              ))}
            </div>
            <div className="flex space-x-2">
              <Button onClick={handleSaveChanges} disabled={savingChanges} data-testid="save-changes-btn">
                {savingChanges ? 'Saving...' : 'Save as Pending Changes'}
              </Button>
              <Button variant="outline" onClick={() => setEditing(false)}>Cancel</Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Pending Changes */}
      {pendingChanges.length > 0 && (
        <Card data-testid="pending-changes">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center">
                <Edit className="h-5 w-5 mr-2" />
                Pending Changes
                <Badge variant="secondary" className="ml-2 bg-yellow-100 text-yellow-800">{pendingChanges.length}</Badge>
              </CardTitle>
              {pendingChanges.some(c => c.status === 'draft') && (
                <Button size="sm" onClick={handleCreateECO} disabled={creatingECO} data-testid="create-eco-btn">
                  <FilePlus className="h-4 w-4 mr-2" />
                  {creatingECO ? 'Creating...' : 'Create ECO from Changes'}
                </Button>
              )}
            </div>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Field</TableHead>
                  <TableHead>Old Value</TableHead>
                  <TableHead>New Value</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>ECO</TableHead>
                  <TableHead></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {pendingChanges.map((change) => (
                  <TableRow key={change.id}>
                    <TableCell className="font-medium">{change.field_name}</TableCell>
                    <TableCell className="text-red-600 line-through">{change.old_value}</TableCell>
                    <TableCell className="text-green-600">{change.new_value}</TableCell>
                    <TableCell>
                      <Badge variant={change.status === 'draft' ? 'secondary' : 'default'}
                        className={change.status === 'pending' ? 'bg-yellow-100 text-yellow-800' : ''}>
                        {change.status}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {change.eco_id ? (
                        <Link to={`/ecos/${change.eco_id}`} className="text-blue-600 hover:underline">
                          {change.eco_id}
                        </Link>
                      ) : '—'}
                    </TableCell>
                    <TableCell>
                      {change.status === 'draft' && (
                        <Button variant="ghost" size="sm" onClick={() => handleDeleteChange(change.id)} data-testid={`delete-change-${change.id}`}>
                          <Trash2 className="h-4 w-4 text-red-500" />
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Part Details */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center">
              <Info className="h-5 w-5 mr-2" />
              Part Details
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="text-sm font-medium text-muted-foreground">IPN</label>
                <p className="font-mono">{part.ipn}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-muted-foreground">Category</label>
                <p className="capitalize">{part.category || 'Unknown'}</p>
              </div>
              {part.manufacturer && (
                <div>
                  <label className="text-sm font-medium text-muted-foreground">Manufacturer</label>
                  <p>{part.manufacturer}</p>
                </div>
              )}
              {part.mpn && (
                <div>
                  <label className="text-sm font-medium text-muted-foreground">MPN</label>
                  <p className="font-mono">{part.mpn}</p>
                </div>
              )}
              {part.stock !== undefined && (
                <div>
                  <label className="text-sm font-medium text-muted-foreground">Stock</label>
                  <p>{part.stock}</p>
                </div>
              )}
              {part.location && (
                <div>
                  <label className="text-sm font-medium text-muted-foreground">Location</label>
                  <p>{part.location}</p>
                </div>
              )}
            </div>
            
            {part.description && (
              <>
                <Separator />
                <div>
                  <label className="text-sm font-medium text-muted-foreground">Description</label>
                  <p className="mt-1">{part.description}</p>
                </div>
              </>
            )}

            {part.notes && (
              <div>
                <label className="text-sm font-medium text-muted-foreground">Notes</label>
                <p className="mt-1 text-sm">{part.notes}</p>
              </div>
            )}

            {part.datasheet && (
              <div>
                <label className="text-sm font-medium text-muted-foreground">Datasheet</label>
                <div className="mt-1">
                  <Button variant="outline" size="sm" asChild>
                    <a href={part.datasheet} target="_blank" rel="noopener noreferrer">
                      View Datasheet
                    </a>
                  </Button>
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Cost Information */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center">
              <DollarSign className="h-5 w-5 mr-2" />
              Cost Information
            </CardTitle>
          </CardHeader>
          <CardContent>
            {costLoading ? (
              <div className="space-y-3">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
                <Skeleton className="h-4 w-1/2" />
              </div>
            ) : (
              <div className="space-y-4">
                {part.cost && (
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">Unit Cost</label>
                    <p className="text-2xl font-bold">${part.cost.toFixed(2)}</p>
                  </div>
                )}
                
                {cost?.last_unit_price && (
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">Last Purchase Price</label>
                    <p className="text-lg font-semibold">${cost.last_unit_price.toFixed(2)}</p>
                    {cost.po_id && (
                      <p className="text-sm text-muted-foreground">
                        PO: {cost.po_id}
                        {cost.last_ordered && (
                          <span> • {new Date(cost.last_ordered).toLocaleDateString()}</span>
                        )}
                      </p>
                    )}
                  </div>
                )}

                {cost?.bom_cost && cost.bom_cost > 0 && (
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">BOM Cost Rollup</label>
                    <p className="text-lg font-semibold">${cost.bom_cost.toFixed(2)}</p>
                    <p className="text-sm text-muted-foreground">
                      Based on latest purchase prices
                    </p>
                  </div>
                )}

                {!cost?.last_unit_price && !part.cost && (
                  <p className="text-muted-foreground">No cost information available</p>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Manufacturers */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center">
              <Factory className="h-5 w-5 mr-2" />
              Manufacturers
              {manufacturers.length > 0 && (
                <Badge variant="secondary" className="ml-2">{manufacturers.length}</Badge>
              )}
            </CardTitle>
            <Button size="sm" onClick={openAddManufacturerDialog} data-testid="add-manufacturer-btn">
              <Plus className="h-4 w-4 mr-2" />
              Add Manufacturer
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {manufacturersLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 2 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : manufacturers.length > 0 ? (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Manufacturer</TableHead>
                    <TableHead>MPN</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Approved</TableHead>
                    <TableHead>Notes</TableHead>
                    <TableHead className="w-[100px]">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {manufacturers.map((mfg) => (
                    <TableRow key={mfg.id} data-testid={`manufacturer-row-${mfg.id}`}>
                      <TableCell className="font-medium">{mfg.manufacturer_name}</TableCell>
                      <TableCell className="font-mono text-sm">{mfg.mpn}</TableCell>
                      <TableCell>
                        {mfg.is_primary ? (
                          <Badge variant="default" className="bg-blue-600" data-testid={`primary-badge-${mfg.id}`}>
                            <Check className="h-3 w-3 mr-1" />
                            Primary
                          </Badge>
                        ) : (
                          <Badge variant="outline">Secondary</Badge>
                        )}
                      </TableCell>
                      <TableCell>
                        {mfg.approved && (
                          <Check className="h-4 w-4 text-green-600" data-testid={`approved-check-${mfg.id}`} />
                        )}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground max-w-[200px] truncate">
                        {mfg.notes || "—"}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center space-x-2">
                          <Button 
                            variant="ghost" 
                            size="sm" 
                            onClick={() => openEditManufacturerDialog(mfg)}
                            data-testid={`edit-manufacturer-${mfg.id}`}
                          >
                            <Edit className="h-4 w-4" />
                          </Button>
                          <Button 
                            variant="ghost" 
                            size="sm" 
                            onClick={() => openDeleteManufacturerDialog(mfg)}
                            data-testid={`delete-manufacturer-${mfg.id}`}
                          >
                            <Trash2 className="h-4 w-4 text-destructive" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              <Factory className="h-8 w-8 mx-auto mb-2 opacity-50" />
              <p>No manufacturers added</p>
              <Button 
                variant="outline" 
                size="sm" 
                className="mt-4"
                onClick={openAddManufacturerDialog}
              >
                <Plus className="h-4 w-4 mr-2" />
                Add Manufacturer
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Add/Edit Manufacturer Dialog */}
      <Dialog open={manufacturerDialogOpen} onOpenChange={setManufacturerDialogOpen}>
        <DialogContent data-testid="manufacturer-dialog">
          <DialogHeader>
            <DialogTitle>
              {editingManufacturer ? "Edit Manufacturer" : "Add Manufacturer"}
            </DialogTitle>
            <DialogDescription>
              {editingManufacturer 
                ? "Update manufacturer information for this part."
                : "Add a new manufacturer for this part."}
            </DialogDescription>
          </DialogHeader>
          
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="manufacturer">Manufacturer *</Label>
              <div className="relative">
                <Input
                  id="manufacturer"
                  value={manufacturerSearch}
                  onChange={(e) => {
                    setManufacturerSearch(e.target.value);
                    setShowManufacturerDropdown(true);
                  }}
                  onFocus={() => setShowManufacturerDropdown(true)}
                  placeholder="Search manufacturers..."
                  data-testid="manufacturer-input"
                  autoComplete="off"
                />
                {showManufacturerDropdown && filteredManufacturers.length > 0 && (
                  <div 
                    className="absolute z-10 w-full mt-1 bg-background border rounded-md shadow-lg max-h-60 overflow-auto"
                    data-testid="manufacturer-dropdown"
                  >
                    {filteredManufacturers.map((mfg) => (
                      <div
                        key={mfg.id}
                        className="px-3 py-2 hover:bg-muted cursor-pointer"
                        onClick={() => handleManufacturerSelect(mfg)}
                        data-testid={`manufacturer-option-${mfg.id}`}
                      >
                        <div className="font-medium">{mfg.name}</div>
                        {mfg.contact_email && (
                          <div className="text-sm text-muted-foreground">{mfg.contact_email}</div>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
              {manufacturerFormErrors.manufacturer && (
                <p className="text-sm text-destructive">{manufacturerFormErrors.manufacturer}</p>
              )}
              {manufacturerForm.manufacturer_id > 0 && (
                <p className="text-sm text-muted-foreground">Selected: {manufacturerForm.manufacturer_name}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="mpn">MPN *</Label>
              <Input
                id="mpn"
                value={manufacturerForm.mpn}
                onChange={(e) => setManufacturerForm({ ...manufacturerForm, mpn: e.target.value })}
                placeholder="e.g., TPS54560ADDAR"
                data-testid="mpn-input"
              />
              {manufacturerFormErrors.mpn && (
                <p className="text-sm text-destructive">{manufacturerFormErrors.mpn}</p>
              )}
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="is_primary"
                checked={manufacturerForm.is_primary}
                onCheckedChange={(checked) => 
                  setManufacturerForm({ ...manufacturerForm, is_primary: checked === true })
                }
                data-testid="primary-checkbox"
              />
              <Label htmlFor="is_primary" className="font-normal">
                Primary Source
              </Label>
            </div>
            {manufacturerFormErrors.is_primary && (
              <p className="text-sm text-destructive">{manufacturerFormErrors.is_primary}</p>
            )}

            <div className="flex items-center space-x-2">
              <Checkbox
                id="approved"
                checked={manufacturerForm.approved}
                onCheckedChange={(checked) => 
                  setManufacturerForm({ ...manufacturerForm, approved: checked === true })
                }
                data-testid="approved-checkbox"
              />
              <Label htmlFor="approved" className="font-normal">
                Approved
              </Label>
            </div>

            <div className="space-y-2">
              <Label htmlFor="notes">Notes</Label>
              <Textarea
                id="notes"
                value={manufacturerForm.notes}
                onChange={(e) => setManufacturerForm({ ...manufacturerForm, notes: e.target.value })}
                placeholder="Optional notes..."
                rows={3}
                data-testid="notes-input"
              />
            </div>
          </div>

          <DialogFooter>
            <Button 
              variant="outline" 
              onClick={() => setManufacturerDialogOpen(false)}
              disabled={savingManufacturer}
            >
              Cancel
            </Button>
            <Button 
              onClick={handleSaveManufacturer} 
              disabled={savingManufacturer}
              data-testid="save-manufacturer-btn"
            >
              {savingManufacturer ? "Saving..." : "Save"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent data-testid="delete-manufacturer-dialog">
          <AlertDialogHeader>
            <AlertDialogTitle>Are you sure?</AlertDialogTitle>
            <AlertDialogDescription asChild>
              <div>
                <p>Are you sure you want to delete this manufacturer?</p>
                {manufacturerToDelete && (
                  <div className="mt-2 p-3 bg-muted rounded-md">
                    <p className="font-medium">{manufacturerToDelete.manufacturer_name}</p>
                    <p className="text-sm font-mono">{manufacturerToDelete.mpn}</p>
                  </div>
                )}
              </div>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deletingManufacturer}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDeleteManufacturer}
              disabled={deletingManufacturer}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              data-testid="confirm-delete-manufacturer"
            >
              {deletingManufacturer ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* BOM Tree for Assemblies */}
      {isAssembly && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center">
                <Layers className="h-5 w-5 mr-2" />
                Bill of Materials
              </CardTitle>
              {!editingBOM && (
                <Button 
                  size="sm" 
                  variant="outline" 
                  onClick={() => setEditingBOM(true)}
                  data-testid="edit-bom-btn"
                >
                  <Edit className="h-4 w-4 mr-2" />
                  Edit BOM
                </Button>
              )}
            </div>
          </CardHeader>
          <CardContent>
            {editingBOM ? (
              <BOMEditor 
                assemblyIPN={ipn!}
                initialItems={bom ? flattenBOMToItems(bom) : []}
                onSave={() => {
                  setEditingBOM(false);
                  fetchBOM();
                  toast.success("BOM updated successfully");
                }}
                onCancel={() => setEditingBOM(false)}
              />
            ) : bomLoading ? (
              <div className="space-y-3">
                {Array.from({ length: 5 }).map((_, i) => (
                  <Skeleton key={i} className="h-8 w-full" />
                ))}
              </div>
            ) : bom ? (
              <div className="border rounded-md p-4">
                <BOMTree node={bom} onPartClick={handleBOMPartClick} gitplmBuildUrl={gitplmConfigured ? gitplmUrl : undefined} />
              </div>
            ) : (
              <div className="text-center py-8 text-muted-foreground">
                <Layers className="h-8 w-8 mx-auto mb-2 opacity-50" />
                <p>No BOM data available for this assembly</p>
                <Button 
                  size="sm" 
                  variant="outline" 
                  className="mt-4"
                  onClick={() => setEditingBOM(true)}
                >
                  <Plus className="h-4 w-4 mr-2" />
                  Create BOM
                </Button>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Market Pricing */}
      {part?.mpn && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center">
                <Store className="h-5 w-5 mr-2" />
                Market Pricing
                {marketPricingCached && (
                  <Badge variant="secondary" className="ml-2 text-xs">Cached</Badge>
                )}
              </CardTitle>
              <Button
                variant="outline"
                size="sm"
                onClick={() => fetchMarketPricing(true)}
                disabled={marketPricingLoading}
                data-testid="refresh-market-pricing"
              >
                <RefreshCw className={`h-4 w-4 mr-1 ${marketPricingLoading ? 'animate-spin' : ''}`} />
                Refresh
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            {marketNotConfigured ? (
              <div className="text-center py-6 text-muted-foreground">
                <Store className="h-8 w-8 mx-auto mb-2 opacity-50" />
                <p className="font-medium">No distributor API keys configured</p>
                <p className="text-sm mt-1">
                  Go to <a href="/settings/distributors" className="underline text-primary">Settings → Distributor API Settings</a> to add your Digikey and/or Mouser API credentials.
                </p>
              </div>
            ) : marketPricingLoading ? (
              <div className="space-y-3">
                {Array.from({ length: 2 }).map((_, i) => (
                  <Skeleton key={i} className="h-24 w-full" />
                ))}
              </div>
            ) : marketPricing.length > 0 ? (
              <div className="space-y-4">
                {marketPricing.map((result, idx) => (
                  <div key={idx} className="border rounded-md p-4">
                    <div className="flex items-center justify-between mb-3">
                      <div>
                        <h4 className="font-semibold">{result.distributor}</h4>
                        <p className="text-sm text-muted-foreground font-mono">{result.distributor_pn}</p>
                      </div>
                      <div className="text-right">
                        <p className="text-sm">
                          Stock: <span className={`font-semibold ${result.stock_qty > 0 ? 'text-green-600' : 'text-red-600'}`}>
                            {result.stock_qty.toLocaleString()}
                          </span>
                        </p>
                        <p className="text-sm text-muted-foreground">
                          Lead time: {result.lead_time_days} days
                        </p>
                      </div>
                    </div>
                    {result.price_breaks && result.price_breaks.length > 0 && (
                      <div className="overflow-x-auto">
                        <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>Qty</TableHead>
                            <TableHead className="text-right">Unit Price</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {result.price_breaks.map((pb, pbIdx) => (
                            <TableRow key={pbIdx}>
                              <TableCell>{pb.qty.toLocaleString()}+</TableCell>
                              <TableCell className="text-right font-mono">
                                ${pb.unit_price.toFixed(4)}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                      </div>
                    )}
                    {result.product_url && (
                      <a
                        href={result.product_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-xs text-blue-600 hover:underline mt-2 inline-block"
                      >
                        View on {result.distributor} →
                      </a>
                    )}
                  </div>
                ))}
              </div>
            ) : marketPricingError ? (
              <div className="text-center py-8 text-muted-foreground">
                <Store className="h-8 w-8 mx-auto mb-2 opacity-50" />
                <p>{marketPricingError}</p>
                {marketPricingUnconfigured.length > 0 && (
                  <p className="mt-2 text-sm">
                    <a href="/distributor-settings" className="text-blue-600 hover:underline">
                      Configure API keys →
                    </a>
                  </p>
                )}
              </div>
            ) : (
              <div className="text-center py-8 text-muted-foreground">
                <Store className="h-8 w-8 mx-auto mb-2 opacity-50" />
                <p>No market pricing available</p>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Where Used */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center">
            <GitBranch className="h-5 w-5 mr-2" />
            Where Used
          </CardTitle>
        </CardHeader>
        <CardContent>
          {whereUsedLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-8 w-full" />
              ))}
            </div>
          ) : whereUsed.length > 0 ? (
            <div className="overflow-x-auto">
              <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Assembly</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead className="text-right">Qty Per</TableHead>
                  <TableHead>Reference</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {whereUsed.map((entry, index) => (
                  <TableRow key={index}>
                    <TableCell>
                      <Link
                        to={`/parts/${encodeURIComponent(entry.assembly_ipn)}`}
                        className="font-mono text-blue-600 hover:underline"
                      >
                        {entry.assembly_ipn}
                      </Link>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {entry.description || "—"}
                    </TableCell>
                    <TableCell className="text-right">{entry.qty}</TableCell>
                    <TableCell>
                      {entry.ref ? (
                        <Badge variant="secondary" className="text-xs">
                          {entry.ref}
                        </Badge>
                      ) : (
                        "—"
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            </div>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              <GitBranch className="h-8 w-8 mx-auto mb-2 opacity-50" />
              <p>This part is not used in any assemblies</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
export default PartDetail;
