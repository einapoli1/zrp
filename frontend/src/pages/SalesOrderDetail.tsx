import { useEffect, useState } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Separator } from "../components/ui/separator";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { ArrowLeft, CheckCircle, Package, Truck, FileText, ClipboardList } from "lucide-react";
import { api, type SalesOrder } from "../lib/api";
import { toast } from "sonner";

const statusSteps = ["draft", "confirmed", "allocated", "picked", "shipped", "invoiced", "closed"];

const statusColors: Record<string, string> = {
  draft: "bg-gray-100 text-gray-800",
  confirmed: "bg-blue-100 text-blue-800",
  allocated: "bg-yellow-100 text-yellow-800",
  picked: "bg-orange-100 text-orange-800",
  shipped: "bg-purple-100 text-purple-800",
  invoiced: "bg-green-100 text-green-800",
  closed: "bg-gray-200 text-gray-600",
};

function SalesOrderDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [order, setOrder] = useState<SalesOrder | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);

  const fetchOrder = async () => {
    if (!id) return;
    try {
      const data = await api.getSalesOrder(id);
      setOrder(data);
    } catch {
      toast.error("Failed to load order");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchOrder(); }, [id]);

  const handleAction = async (action: string) => {
    if (!id) return;
    setActionLoading(true);
    try {
      let result: SalesOrder;
      switch (action) {
        case "confirm": result = await api.confirmSalesOrder(id); break;
        case "allocate": result = await api.allocateSalesOrder(id); break;
        case "pick": result = await api.pickSalesOrder(id); break;
        case "ship": result = await api.shipSalesOrder(id); break;
        case "invoice": result = await api.invoiceSalesOrder(id); break;
        default: return;
      }
      setOrder(result);
      toast.success(`Order ${action}ed successfully`);
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : "Action failed";
      toast.error(message);
    } finally {
      setActionLoading(false);
    }
  };

  if (loading) return <div className="p-8">Loading...</div>;
  if (!order) return <div className="p-8">Order not found</div>;

  const currentStep = statusSteps.indexOf(order.status);
  const total = (order.lines || []).reduce((sum, l) => sum + l.qty * l.unit_price, 0);

  const nextAction = () => {
    switch (order.status) {
      case "draft": return { label: "Confirm Order", action: "confirm", icon: CheckCircle };
      case "confirmed": return { label: "Allocate Inventory", action: "allocate", icon: Package };
      case "allocated": return { label: "Pick Order", action: "pick", icon: ClipboardList };
      case "picked": return { label: "Ship Order", action: "ship", icon: Truck };
      case "shipped": return { label: "Create Invoice", action: "invoice", icon: FileText };
      default: return null;
    }
  };

  const next = nextAction();

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate("/sales-orders")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div>
          <h1 className="text-2xl font-bold">{order.id}</h1>
          <p className="text-muted-foreground">{order.customer}</p>
        </div>
        <Badge className={statusColors[order.status] || ""} data-testid="order-status">
          {order.status}
        </Badge>
      </div>

      {/* Status progression bar */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center justify-between mb-4">
            {statusSteps.map((step, i) => (
              <div key={step} className="flex items-center flex-1">
                <div className={`flex items-center justify-center w-8 h-8 rounded-full text-xs font-bold ${
                  i <= currentStep ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"
                }`}>
                  {i + 1}
                </div>
                <span className={`ml-2 text-xs ${i <= currentStep ? "font-medium" : "text-muted-foreground"}`}>
                  {step.charAt(0).toUpperCase() + step.slice(1)}
                </span>
                {i < statusSteps.length - 1 && (
                  <div className={`flex-1 h-0.5 mx-2 ${i < currentStep ? "bg-primary" : "bg-muted"}`} />
                )}
              </div>
            ))}
          </div>
          {next && (
            <div className="flex justify-end">
              <Button onClick={() => handleAction(next.action)} disabled={actionLoading} data-testid="next-action">
                <next.icon className="h-4 w-4 mr-2" />
                {actionLoading ? "Processing..." : next.label}
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Order details */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <Card>
          <CardHeader><CardTitle>Details</CardTitle></CardHeader>
          <CardContent className="space-y-2">
            <div className="flex justify-between"><span className="text-muted-foreground">Customer</span><span>{order.customer}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">Quote</span>
              {order.quote_id ? (
                <Link 
                  to={`/quotes/${order.quote_id}`} 
                  className="font-mono text-blue-600 hover:underline focus-visible:ring-2 focus-visible:ring-ring focus-visible:rounded"
                >
                  {order.quote_id}
                </Link>
              ) : <span>—</span>}
            </div>
            <div className="flex justify-between"><span className="text-muted-foreground">Created By</span><span>{order.created_by || "—"}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">Created</span><span>{new Date(order.created_at).toLocaleString()}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">Updated</span><span>{new Date(order.updated_at).toLocaleString()}</span></div>
            {order.shipment_id && (
              <div className="flex justify-between"><span className="text-muted-foreground">Shipment</span>
                <Link 
                  to={`/shipments/${order.shipment_id}`}
                  className="font-mono text-blue-600 hover:underline focus-visible:ring-2 focus-visible:ring-ring focus-visible:rounded"
                >
                  {order.shipment_id}
                </Link>
              </div>
            )}
            {order.invoice_id && (
              <div className="flex justify-between"><span className="text-muted-foreground">Invoice</span><span className="font-mono">{order.invoice_id}</span></div>
            )}
            {order.notes && (
              <>
                <Separator />
                <p className="text-sm">{order.notes}</p>
              </>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle>Summary</CardTitle></CardHeader>
          <CardContent className="space-y-2">
            <div className="flex justify-between"><span className="text-muted-foreground">Line Items</span><span>{order.lines?.length || 0}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">Total Qty</span><span>{(order.lines || []).reduce((s, l) => s + l.qty, 0)}</span></div>
            <Separator />
            <div className="flex justify-between text-lg font-bold"><span>Total</span><span>${total.toLocaleString(undefined, { minimumFractionDigits: 2 })}</span></div>
          </CardContent>
        </Card>
      </div>

      {/* Lines */}
      <Card>
        <CardHeader><CardTitle>Order Lines</CardTitle></CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>IPN</TableHead>
                <TableHead>Description</TableHead>
                <TableHead className="text-right">Qty</TableHead>
                <TableHead className="text-right">Allocated</TableHead>
                <TableHead className="text-right">Picked</TableHead>
                <TableHead className="text-right">Shipped</TableHead>
                <TableHead className="text-right">Unit Price</TableHead>
                <TableHead className="text-right">Line Total</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(order.lines || []).map((line) => (
                <TableRow key={line.id}>
                  <TableCell className="font-mono">{line.ipn}</TableCell>
                  <TableCell>{line.description}</TableCell>
                  <TableCell className="text-right">{line.qty}</TableCell>
                  <TableCell className="text-right">{line.qty_allocated}</TableCell>
                  <TableCell className="text-right">{line.qty_picked}</TableCell>
                  <TableCell className="text-right">{line.qty_shipped}</TableCell>
                  <TableCell className="text-right">${line.unit_price.toFixed(2)}</TableCell>
                  <TableCell className="text-right font-medium">${(line.qty * line.unit_price).toFixed(2)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}

export default SalesOrderDetail;
