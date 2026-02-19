import { useEffect, useState } from "react";
import {
  DollarSign,
  Plus,
  TrendingUp,
  TrendingDown,
  Pencil,
  Trash2,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "../components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../components/ui/table";
import { Checkbox } from "../components/ui/checkbox";
import { api, type ProductPricing, type CostAnalysis } from "../lib/api";

function marginColor(pct: number): string {
  if (pct < 15) return "text-red-600 bg-red-50";
  if (pct <= 30) return "text-yellow-600 bg-yellow-50";
  return "text-green-600 bg-green-50";
}

function marginBadgeVariant(pct: number): "destructive" | "secondary" | "default" {
  if (pct < 15) return "destructive";
  if (pct <= 30) return "secondary";
  return "default";
}

interface PricingSummary {
  product_ipn: string;
  standard_price: number;
  total_cost: number;
  margin_pct: number;
  tier_count: number;
}

export default function Pricing() {
  const [pricing, setPricing] = useState<ProductPricing[]>([]);
  const [analysis, setAnalysis] = useState<CostAnalysis[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [showEdit, setShowEdit] = useState(false);
  const [editItem, setEditItem] = useState<ProductPricing | null>(null);
  const [showBulk, setShowBulk] = useState(false);
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const [search, setSearch] = useState("");
  const [bulkType, setBulkType] = useState("percentage");
  const [bulkValue, setBulkValue] = useState("");
  const [form, setForm] = useState({
    product_ipn: "",
    pricing_tier: "standard",
    min_qty: "1",
    max_qty: "100",
    unit_price: "",
    currency: "USD",
    effective_date: "",
    expiry_date: "",
    notes: "",
  });

  useEffect(() => {
    loadData();
  }, []);

  async function loadData() {
    try {
      const [pricingData, analysisData] = await Promise.all([
        api.getProductPricing(),
        api.getCostAnalysis(),
      ]);
      setPricing(pricingData);
      setAnalysis(analysisData);
    } catch (err) {
      console.error("Failed to load pricing data:", err);
    } finally {
      setLoading(false);
    }
  }

  // Build summary by product
  const summaryMap = new Map<string, PricingSummary>();
  for (const p of pricing) {
    if (!summaryMap.has(p.product_ipn)) {
      const ca = analysis.find((a) => a.product_ipn === p.product_ipn);
      summaryMap.set(p.product_ipn, {
        product_ipn: p.product_ipn,
        standard_price: 0,
        total_cost: ca?.total_cost ?? 0,
        margin_pct: ca?.margin_pct ?? 0,
        tier_count: 0,
      });
    }
    const s = summaryMap.get(p.product_ipn)!;
    s.tier_count++;
    if (p.pricing_tier === "standard") {
      s.standard_price = p.unit_price;
    }
  }
  const summaries = Array.from(summaryMap.values()).filter(
    (s) => !search || s.product_ipn.toLowerCase().includes(search.toLowerCase())
  );

  async function handleCreate() {
    try {
      await api.createProductPricing({
        product_ipn: form.product_ipn,
        pricing_tier: form.pricing_tier,
        min_qty: parseInt(form.min_qty) || 0,
        max_qty: parseInt(form.max_qty) || 0,
        unit_price: parseFloat(form.unit_price) || 0,
        currency: form.currency,
        effective_date: form.effective_date,
        expiry_date: form.expiry_date,
        notes: form.notes,
      });
      setShowCreate(false);
      resetForm();
      loadData();
    } catch (err) {
      console.error("Failed to create:", err);
    }
  }

  async function handleUpdate() {
    if (!editItem) return;
    try {
      await api.updateProductPricing(editItem.id, {
        product_ipn: form.product_ipn,
        pricing_tier: form.pricing_tier,
        min_qty: parseInt(form.min_qty) || 0,
        max_qty: parseInt(form.max_qty) || 0,
        unit_price: parseFloat(form.unit_price) || 0,
        currency: form.currency,
        effective_date: form.effective_date,
        expiry_date: form.expiry_date,
        notes: form.notes,
      });
      setShowEdit(false);
      setEditItem(null);
      resetForm();
      loadData();
    } catch (err) {
      console.error("Failed to update:", err);
    }
  }

  async function handleDelete(id: number) {
    try {
      await api.deleteProductPricing(id);
      loadData();
    } catch (err) {
      console.error("Failed to delete:", err);
    }
  }

  async function handleBulkUpdate() {
    if (selectedIds.length === 0) return;
    try {
      await api.bulkUpdateProductPricing(selectedIds, bulkType, parseFloat(bulkValue) || 0);
      setShowBulk(false);
      setSelectedIds([]);
      setBulkValue("");
      loadData();
    } catch (err) {
      console.error("Failed to bulk update:", err);
    }
  }

  function openEdit(p: ProductPricing) {
    setEditItem(p);
    setForm({
      product_ipn: p.product_ipn,
      pricing_tier: p.pricing_tier,
      min_qty: String(p.min_qty),
      max_qty: String(p.max_qty),
      unit_price: String(p.unit_price),
      currency: p.currency,
      effective_date: p.effective_date,
      expiry_date: p.expiry_date || "",
      notes: p.notes || "",
    });
    setShowEdit(true);
  }

  function resetForm() {
    setForm({
      product_ipn: "",
      pricing_tier: "standard",
      min_qty: "1",
      max_qty: "100",
      unit_price: "",
      currency: "USD",
      effective_date: "",
      expiry_date: "",
      notes: "",
    });
  }

  function toggleSelect(id: number) {
    setSelectedIds((prev) =>
      prev.includes(id) ? prev.filter((i) => i !== id) : [...prev, id]
    );
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <p className="text-muted-foreground">Loading pricing data...</p>
      </div>
    );
  }

  const totalProducts = summaries.length;
  const avgMargin = summaries.length > 0
    ? summaries.reduce((a, s) => a + s.margin_pct, 0) / summaries.length
    : 0;
  const lowMarginCount = summaries.filter((s) => s.margin_pct < 15).length;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Pricing</h1>
          <p className="text-muted-foreground">
            Manage product pricing, cost analysis, and margin tracking.
          </p>
        </div>
        <div className="flex gap-2">
          {selectedIds.length > 0 && (
            <Button variant="outline" onClick={() => setShowBulk(true)}>
              Bulk Update ({selectedIds.length})
            </Button>
          )}
          <Button onClick={() => { resetForm(); setShowCreate(true); }}>
            <Plus className="mr-2 h-4 w-4" />
            Add Pricing
          </Button>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Products Priced</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalProducts}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Average Margin</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{avgMargin.toFixed(1)}%</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Low Margin Items</CardTitle>
            <TrendingDown className="h-4 w-4 text-red-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-600">{lowMarginCount}</div>
          </CardContent>
        </Card>
      </div>

      <Input
        placeholder="Search by IPN..."
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="max-w-sm"
      />

      <Tabs defaultValue="pricing">
        <TabsList>
          <TabsTrigger value="pricing">Pricing Tiers</TabsTrigger>
          <TabsTrigger value="analysis">Cost Analysis</TabsTrigger>
        </TabsList>

        <TabsContent value="pricing" className="space-y-4">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-8"></TableHead>
                <TableHead>Product IPN</TableHead>
                <TableHead>Tier</TableHead>
                <TableHead>Qty Range</TableHead>
                <TableHead className="text-right">Unit Price</TableHead>
                <TableHead>Effective</TableHead>
                <TableHead className="text-right">Margin</TableHead>
                <TableHead></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {pricing
                .filter(
                  (p) =>
                    !search || p.product_ipn.toLowerCase().includes(search.toLowerCase())
                )
                .map((p) => {
                  const ca = analysis.find((a) => a.product_ipn === p.product_ipn);
                  const margin = ca?.margin_pct ?? 0;
                  return (
                    <TableRow key={p.id}>
                      <TableCell>
                        <Checkbox
                          checked={selectedIds.includes(p.id)}
                          onCheckedChange={() => toggleSelect(p.id)}
                        />
                      </TableCell>
                      <TableCell className="font-medium">{p.product_ipn}</TableCell>
                      <TableCell>
                        <Badge variant="outline">{p.pricing_tier}</Badge>
                      </TableCell>
                      <TableCell>
                        {p.min_qty} – {p.max_qty}
                      </TableCell>
                      <TableCell className="text-right font-mono">
                        ${p.unit_price.toFixed(2)}
                      </TableCell>
                      <TableCell>{p.effective_date}</TableCell>
                      <TableCell className="text-right">
                        <span
                          className={`inline-flex items-center rounded-full px-2 py-1 text-xs font-medium ${marginColor(margin)}`}
                        >
                          {margin.toFixed(1)}%
                        </span>
                      </TableCell>
                      <TableCell>
                        <div className="flex gap-1">
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => openEdit(p)}
                          >
                            <Pencil className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => handleDelete(p.id)}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  );
                })}
            </TableBody>
          </Table>
        </TabsContent>

        <TabsContent value="analysis" className="space-y-4">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Product IPN</TableHead>
                <TableHead className="text-right">BOM Cost</TableHead>
                <TableHead className="text-right">Labor</TableHead>
                <TableHead className="text-right">Overhead</TableHead>
                <TableHead className="text-right">Total Cost</TableHead>
                <TableHead className="text-right">Selling Price</TableHead>
                <TableHead className="text-right">Margin</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {analysis
                .filter(
                  (a) =>
                    !search || a.product_ipn.toLowerCase().includes(search.toLowerCase())
                )
                .map((a) => (
                  <TableRow key={a.id}>
                    <TableCell className="font-medium">{a.product_ipn}</TableCell>
                    <TableCell className="text-right font-mono">
                      ${a.bom_cost.toFixed(2)}
                    </TableCell>
                    <TableCell className="text-right font-mono">
                      ${a.labor_cost.toFixed(2)}
                    </TableCell>
                    <TableCell className="text-right font-mono">
                      ${a.overhead_cost.toFixed(2)}
                    </TableCell>
                    <TableCell className="text-right font-mono">
                      ${a.total_cost.toFixed(2)}
                    </TableCell>
                    <TableCell className="text-right font-mono">
                      ${a.selling_price.toFixed(2)}
                    </TableCell>
                    <TableCell className="text-right">
                      <Badge variant={marginBadgeVariant(a.margin_pct)}>
                        {a.margin_pct.toFixed(1)}%
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))}
            </TableBody>
          </Table>
        </TabsContent>
      </Tabs>

      {/* Create Dialog */}
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create Pricing</DialogTitle>
          </DialogHeader>
          <PricingForm form={form} setForm={setForm} />
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreate(false)}>Cancel</Button>
            <Button onClick={handleCreate}>Create</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog open={showEdit} onOpenChange={setShowEdit}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Pricing</DialogTitle>
          </DialogHeader>
          <PricingForm form={form} setForm={setForm} />
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowEdit(false)}>Cancel</Button>
            <Button onClick={handleUpdate}>Save</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Bulk Update Dialog */}
      <Dialog open={showBulk} onOpenChange={setShowBulk}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Bulk Price Update</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Updating {selectedIds.length} pricing entries
            </p>
            <div className="space-y-2">
              <Label>Adjustment Type</Label>
              <Select value={bulkType} onValueChange={setBulkType}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="percentage">Percentage (%)</SelectItem>
                  <SelectItem value="absolute">Absolute ($)</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Value</Label>
              <Input
                type="number"
                value={bulkValue}
                onChange={(e) => setBulkValue(e.target.value)}
                placeholder={bulkType === "percentage" ? "e.g. 10 for +10%" : "e.g. 1.50"}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowBulk(false)}>Cancel</Button>
            <Button onClick={handleBulkUpdate}>Apply</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function PricingForm({
  form,
  setForm,
}: {
  form: { product_ipn: string; pricing_tier: string; min_qty: string; max_qty: string; unit_price: string; currency: string; effective_date: string; expiry_date: string; notes: string };
  setForm: React.Dispatch<React.SetStateAction<{ product_ipn: string; pricing_tier: string; min_qty: string; max_qty: string; unit_price: string; currency: string; effective_date: string; expiry_date: string; notes: string }>>;
}) {
  const update = (field: string, value: string) =>
    setForm({ ...form, [field]: value });

  return (
    <div className="grid gap-4 py-4">
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label>Product IPN</Label>
          <Input
            value={form.product_ipn}
            onChange={(e) => update("product_ipn", e.target.value)}
            placeholder="IPN-001"
          />
        </div>
        <div className="space-y-2">
          <Label>Pricing Tier</Label>
          <Select
            value={form.pricing_tier}
            onValueChange={(v) => update("pricing_tier", v)}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="standard">Standard</SelectItem>
              <SelectItem value="volume">Volume</SelectItem>
              <SelectItem value="distributor">Distributor</SelectItem>
              <SelectItem value="oem">OEM</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>
      <div className="grid grid-cols-3 gap-4">
        <div className="space-y-2">
          <Label>Min Qty</Label>
          <Input
            type="number"
            value={form.min_qty}
            onChange={(e) => update("min_qty", e.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label>Max Qty</Label>
          <Input
            type="number"
            value={form.max_qty}
            onChange={(e) => update("max_qty", e.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label>Unit Price</Label>
          <Input
            type="number"
            step="0.01"
            value={form.unit_price}
            onChange={(e) => update("unit_price", e.target.value)}
          />
        </div>
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label>Effective Date</Label>
          <Input
            type="date"
            value={form.effective_date}
            onChange={(e) => update("effective_date", e.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label>Expiry Date</Label>
          <Input
            type="date"
            value={form.expiry_date}
            onChange={(e) => update("expiry_date", e.target.value)}
          />
        </div>
      </div>
      <div className="space-y-2">
        <Label>Notes</Label>
        <Input
          value={form.notes}
          onChange={(e) => update("notes", e.target.value)}
          placeholder="Optional notes"
        />
      </div>
    </div>
  );
}
