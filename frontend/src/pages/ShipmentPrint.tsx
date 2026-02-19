import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { api, type Shipment, type PackList } from "../lib/api";

function ShipmentPrint() {
  const { id } = useParams<{ id: string }>();
  const [shipment, setShipment] = useState<Shipment | null>(null);
  const [packList, setPackList] = useState<PackList | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!id) return;
    const fetchData = async () => {
      try {
        const [s, pl] = await Promise.all([
          api.getShipment(id),
          api.getShipmentPackList(id),
        ]);
        setShipment(s);
        setPackList(pl);
      } catch (error) {
        console.error("Failed to load pack list:", error);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [id]);

  if (loading) return <div className="p-8">Loading...</div>;
  if (!shipment || !packList) return <div className="p-8">Not found</div>;

  return (
    <div className="p-8 max-w-3xl mx-auto print-friendly">
      <style>{`
        @media print {
          body { margin: 0; }
          .no-print { display: none !important; }
        }
        .print-friendly { font-family: sans-serif; }
      `}</style>

      <div className="no-print mb-4">
        <button onClick={() => window.print()} className="px-4 py-2 bg-blue-600 text-white rounded">
          Print Pack List
        </button>
        <button onClick={() => window.history.back()} className="ml-2 px-4 py-2 border rounded">
          Back
        </button>
      </div>

      <h1 className="text-2xl font-bold mb-2">Pack List</h1>
      <div className="mb-4 text-sm">
        <div><strong>Shipment:</strong> {shipment.id}</div>
        <div><strong>Type:</strong> {shipment.type}</div>
        <div><strong>Carrier:</strong> {shipment.carrier || "—"}</div>
        <div><strong>Tracking:</strong> {shipment.tracking_number || "—"}</div>
        <div><strong>From:</strong> {shipment.from_address}</div>
        <div><strong>To:</strong> {shipment.to_address}</div>
        <div><strong>Date:</strong> {new Date(packList.created_at).toLocaleDateString()}</div>
      </div>

      <table className="w-full border-collapse border text-sm">
        <thead>
          <tr className="bg-gray-100">
            <th className="border p-2 text-left">#</th>
            <th className="border p-2 text-left">IPN</th>
            <th className="border p-2 text-left">Serial Number</th>
            <th className="border p-2 text-right">Qty</th>
            <th className="border p-2 text-left">Work Order</th>
            <th className="border p-2 text-left">RMA</th>
          </tr>
        </thead>
        <tbody>
          {(packList.lines || []).map((line, idx) => (
            <tr key={line.id}>
              <td className="border p-2">{idx + 1}</td>
              <td className="border p-2">{line.ipn || "—"}</td>
              <td className="border p-2">{line.serial_number || "—"}</td>
              <td className="border p-2 text-right">{line.qty}</td>
              <td className="border p-2">{line.work_order_id || "—"}</td>
              <td className="border p-2">{line.rma_id || "—"}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <div className="mt-8 text-sm text-gray-500">
        <div>Total items: {(packList.lines || []).reduce((sum, l) => sum + l.qty, 0)}</div>
        <div>Pack list generated: {new Date(packList.created_at).toLocaleString()}</div>
      </div>
    </div>
  );
}

export default ShipmentPrint;
