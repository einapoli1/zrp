import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { Label } from "../components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { Truck, Plus } from "lucide-react";
import { api, type Shipment } from "../lib/api";

function Shipments() {
  const navigate = useNavigate();
  const [shipments, setShipments] = useState<Shipment[]>([]);
  const [loading, setLoading] = useState(true);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [formData, setFormData] = useState({
    type: "outbound",
    carrier: "",
    tracking_number: "",
    from_address: "",
    to_address: "",
    notes: "",
  });

  useEffect(() => {
    const fetchShipments = async () => {
      try {
        const data = await api.getShipments();
        setShipments(data);
      } catch (error) {
        console.error("Failed to fetch shipments:", error);
      } finally {
        setLoading(false);
      }
    };
    fetchShipments();
  }, []);

  const handleCreateShipment = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const newShipment = await api.createShipment(formData);
      setShipments([newShipment, ...shipments]);
      setCreateDialogOpen(false);
      setFormData({ type: "outbound", carrier: "", tracking_number: "", from_address: "", to_address: "", notes: "" });
    } catch (error) {
      console.error("Failed to create shipment:", error);
    }
  };

  const getStatusBadgeVariant = (status: string) => {
    switch (status) {
      case "delivered": return "default";
      case "shipped": return "secondary";
      case "packed": return "outline";
      default: return "destructive";
    }
  };

  const getTypeBadgeVariant = (type: string) => {
    return type === "outbound" ? "default" : "secondary";
  };

  if (loading) return <div className="p-6">Loading shipments...</div>;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Shipments</h1>
          <p className="text-muted-foreground">Manage shipping and logistics</p>
        </div>
        <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button><Plus className="mr-2 h-4 w-4" />Create Shipment</Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader><DialogTitle>Create Shipment</DialogTitle></DialogHeader>
            <form onSubmit={handleCreateShipment} className="space-y-4">
              <div>
                <Label htmlFor="type">Type</Label>
                <Select value={formData.type} onValueChange={(v) => setFormData({ ...formData, type: v })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="outbound">Outbound</SelectItem>
                    <SelectItem value="inbound">Inbound</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div>
                <Label htmlFor="carrier">Carrier</Label>
                <Input id="carrier" value={formData.carrier} onChange={(e) => setFormData({ ...formData, carrier: e.target.value })} placeholder="FedEx, UPS, DHL..." />
              </div>
              <div>
                <Label htmlFor="tracking">Tracking Number</Label>
                <Input id="tracking" value={formData.tracking_number} onChange={(e) => setFormData({ ...formData, tracking_number: e.target.value })} />
              </div>
              <div>
                <Label htmlFor="from">From Address</Label>
                <Input id="from" value={formData.from_address} onChange={(e) => setFormData({ ...formData, from_address: e.target.value })} />
              </div>
              <div>
                <Label htmlFor="to">To Address</Label>
                <Input id="to" value={formData.to_address} onChange={(e) => setFormData({ ...formData, to_address: e.target.value })} />
              </div>
              <div>
                <Label htmlFor="notes">Notes</Label>
                <Textarea id="notes" value={formData.notes} onChange={(e) => setFormData({ ...formData, notes: e.target.value })} />
              </div>
              <Button type="submit" className="w-full">Create</Button>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <Card>
        <CardHeader><CardTitle className="flex items-center gap-2"><Truck className="h-5 w-5" />All Shipments</CardTitle></CardHeader>
        <CardContent>
          {shipments.length === 0 ? (
            <p className="text-center text-muted-foreground py-8">No shipments found. Create your first shipment to get started.</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Carrier</TableHead>
                  <TableHead>Tracking</TableHead>
                  <TableHead>To</TableHead>
                  <TableHead>Created</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {shipments.map((s) => (
                  <TableRow key={s.id} className="cursor-pointer" onClick={() => navigate(`/shipments/${s.id}`)}>
                    <TableCell className="font-medium">{s.id}</TableCell>
                    <TableCell><Badge variant={getTypeBadgeVariant(s.type)}>{s.type}</Badge></TableCell>
                    <TableCell><Badge variant={getStatusBadgeVariant(s.status)}>{s.status}</Badge></TableCell>
                    <TableCell>{s.carrier || "—"}</TableCell>
                    <TableCell>{s.tracking_number || "—"}</TableCell>
                    <TableCell>{s.to_address || "—"}</TableCell>
                    <TableCell>{new Date(s.created_at).toLocaleDateString()}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

export default Shipments;
