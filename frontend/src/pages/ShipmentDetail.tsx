import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "../components/ui/dialog";
import { Truck, Package, Printer, ArrowLeft } from "lucide-react";
import { api, type Shipment } from "../lib/api";

function ShipmentDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [shipment, setShipment] = useState<Shipment | null>(null);
  const [loading, setLoading] = useState(true);
  const [shipDialogOpen, setShipDialogOpen] = useState(false);
  const [trackingNumber, setTrackingNumber] = useState("");
  const [carrier, setCarrier] = useState("");

  useEffect(() => {
    if (!id) return;
    const fetchShipment = async () => {
      try {
        const data = await api.getShipment(id);
        setShipment(data);
      } catch (error) {
        console.error("Failed to fetch shipment:", error);
      } finally {
        setLoading(false);
      }
    };
    fetchShipment();
  }, [id]);

  const handleShip = async () => {
    if (!id) return;
    try {
      const updated = await api.shipShipment(id, trackingNumber, carrier);
      setShipment(updated);
      setShipDialogOpen(false);
    } catch (error) {
      console.error("Failed to ship:", error);
    }
  };

  const handleDeliver = async () => {
    if (!id) return;
    try {
      const updated = await api.deliverShipment(id);
      setShipment(updated);
    } catch (error) {
      console.error("Failed to deliver:", error);
    }
  };

  if (loading) return <div className="p-6">Loading shipment...</div>;
  if (!shipment) return <div className="p-6">Shipment not found</div>;

  const canShip = shipment.status === "draft" || shipment.status === "packed";
  const canDeliver = shipment.status === "shipped";

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="sm" onClick={() => navigate("/shipments")}>
          <ArrowLeft className="h-4 w-4 mr-1" />Back
        </Button>
        <div className="flex-1">
          <h1 className="text-3xl font-bold tracking-tight">{shipment.id}</h1>
          <div className="flex gap-2 mt-1">
            <Badge>{shipment.type}</Badge>
            <Badge variant="secondary">{shipment.status}</Badge>
          </div>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => navigate(`/shipments/${id}/print`)}>
            <Printer className="h-4 w-4 mr-1" />Pack List
          </Button>
          {canShip && (
            <Dialog open={shipDialogOpen} onOpenChange={setShipDialogOpen}>
              <DialogTrigger asChild>
                <Button><Truck className="h-4 w-4 mr-1" />Mark Shipped</Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader><DialogTitle>Ship {shipment.id}</DialogTitle></DialogHeader>
                <div className="space-y-4">
                  <div>
                    <Label>Carrier</Label>
                    <Input value={carrier} onChange={(e) => setCarrier(e.target.value)} placeholder="FedEx, UPS, DHL..." />
                  </div>
                  <div>
                    <Label>Tracking Number</Label>
                    <Input value={trackingNumber} onChange={(e) => setTrackingNumber(e.target.value)} placeholder="1Z999..." />
                  </div>
                  <Button onClick={handleShip} className="w-full">Confirm Ship</Button>
                </div>
              </DialogContent>
            </Dialog>
          )}
          {canDeliver && (
            <Button onClick={handleDeliver} variant="default">
              <Package className="h-4 w-4 mr-1" />Mark Delivered
            </Button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <Card>
          <CardHeader><CardTitle>Shipment Details</CardTitle></CardHeader>
          <CardContent className="space-y-2">
            <div><span className="text-muted-foreground">Carrier:</span> {shipment.carrier || "—"}</div>
            <div><span className="text-muted-foreground">Tracking:</span> {shipment.tracking_number || "—"}</div>
            <div><span className="text-muted-foreground">From:</span> {shipment.from_address || "—"}</div>
            <div><span className="text-muted-foreground">To:</span> {shipment.to_address || "—"}</div>
            <div><span className="text-muted-foreground">Ship Date:</span> {shipment.ship_date ? new Date(shipment.ship_date).toLocaleDateString() : "—"}</div>
            <div><span className="text-muted-foreground">Delivery Date:</span> {shipment.delivery_date ? new Date(shipment.delivery_date).toLocaleDateString() : "—"}</div>
            <div><span className="text-muted-foreground">Notes:</span> {shipment.notes || "—"}</div>
            <div><span className="text-muted-foreground">Created by:</span> {shipment.created_by}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle>Line Items</CardTitle></CardHeader>
          <CardContent>
            {(!shipment.lines || shipment.lines.length === 0) ? (
              <p className="text-muted-foreground">No line items</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>IPN</TableHead>
                    <TableHead>Serial</TableHead>
                    <TableHead>Qty</TableHead>
                    <TableHead>WO</TableHead>
                    <TableHead>RMA</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {shipment.lines.map((line) => (
                    <TableRow key={line.id}>
                      <TableCell>{line.ipn || "—"}</TableCell>
                      <TableCell>{line.serial_number || "—"}</TableCell>
                      <TableCell>{line.qty}</TableCell>
                      <TableCell>{line.work_order_id || "—"}</TableCell>
                      <TableCell>{line.rma_id || "—"}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

export default ShipmentDetail;
