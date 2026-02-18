import { BrowserRouter as Router, Routes, Route } from "react-router-dom";
import { AppLayout } from "./layouts/AppLayout";
import { Dashboard } from "./pages/Dashboard";
import { Inventory } from "./pages/Inventory";
import { InventoryDetail } from "./pages/InventoryDetail";
import { Procurement } from "./pages/Procurement";
import { PODetail } from "./pages/PODetail";
import { Vendors } from "./pages/Vendors";
import { VendorDetail } from "./pages/VendorDetail";
import { WorkOrders } from "./pages/WorkOrders";
import { WorkOrderDetail } from "./pages/WorkOrderDetail";
import { NCRs } from "./pages/NCRs";
import { NCRDetail } from "./pages/NCRDetail";
import { RMAs } from "./pages/RMAs";
import { RMADetail } from "./pages/RMADetail";
import { Testing } from "./pages/Testing";
import { Devices } from "./pages/Devices";
import { DeviceDetail } from "./pages/DeviceDetail";
import { Firmware } from "./pages/Firmware";
import { FirmwareDetail } from "./pages/FirmwareDetail";
import { Quotes } from "./pages/Quotes";
import { QuoteDetail } from "./pages/QuoteDetail";
import { Parts } from "./pages/Parts";
import { PartDetail } from "./pages/PartDetail";
import { ECOs } from "./pages/ECOs";
import { ECODetail } from "./pages/ECODetail";
import { Documents } from "./pages/Documents";
import { Calendar } from "./pages/Calendar";
import { Reports } from "./pages/Reports";
import { Audit } from "./pages/Audit";
import { Users } from "./pages/Users";
import { APIKeys } from "./pages/APIKeys";
import { EmailSettings } from "./pages/EmailSettings";

// Placeholder components for other pages
const PlaceholderPage = ({ title }: { title: string }) => (
  <div className="space-y-6">
    <div>
      <h1 className="text-3xl font-bold tracking-tight">{title}</h1>
      <p className="text-muted-foreground">
        This page is under construction. The React foundation is ready - 
        individual module pages can be built on top of this structure.
      </p>
    </div>
    <div className="rounded-lg border border-dashed p-8 text-center">
      <h3 className="text-lg font-semibold">Coming Soon</h3>
      <p className="text-sm text-muted-foreground mt-2">
        This {title.toLowerCase()} interface will be implemented next.
      </p>
    </div>
  </div>
);

function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<AppLayout />}>
          <Route index element={<Dashboard />} />
          
          {/* Engineering */}
          <Route path="/parts" element={<Parts />} />
          <Route path="/parts/:ipn" element={<PartDetail />} />
          <Route path="/ecos" element={<ECOs />} />
          <Route path="/ecos/:id" element={<ECODetail />} />
          <Route path="/documents" element={<Documents />} />
          <Route path="/testing" element={<Testing />} />
          <Route path="/ncrs" element={<NCRs />} />
          <Route path="/ncrs/:id" element={<NCRDetail />} />
          <Route path="/rmas" element={<RMAs />} />
          <Route path="/rmas/:id" element={<RMADetail />} />
          <Route path="/devices" element={<Devices />} />
          <Route path="/devices/:serialNumber" element={<DeviceDetail />} />
          <Route path="/firmware" element={<Firmware />} />
          <Route path="/firmware/:id" element={<FirmwareDetail />} />
          <Route path="/quotes" element={<Quotes />} />
          <Route path="/quotes/:id" element={<QuoteDetail />} />
          
          {/* Supply Chain */}
          <Route path="/vendors" element={<Vendors />} />
          <Route path="/vendors/:id" element={<VendorDetail />} />
          <Route path="/purchase-orders" element={<Procurement />} />
          <Route path="/purchase-orders/:id" element={<PODetail />} />
          <Route path="/procurement" element={<Procurement />} />
          
          {/* Manufacturing */}
          <Route path="/work-orders" element={<WorkOrders />} />
          <Route path="/work-orders/:id" element={<WorkOrderDetail />} />
          <Route path="/inventory" element={<Inventory />} />
          <Route path="/inventory/:ipn" element={<InventoryDetail />} />
          
          {/* Field & Service */}
          <Route path="/field-reports" element={<PlaceholderPage title="Field Reports" />} />
          <Route path="/pricing" element={<PlaceholderPage title="Pricing" />} />
          
          {/* Reports */}
          <Route path="/reports" element={<Reports />} />
          <Route path="/calendar" element={<Calendar />} />
          
          {/* Admin */}
          <Route path="/users" element={<Users />} />
          <Route path="/audit" element={<Audit />} />
          <Route path="/api-keys" element={<APIKeys />} />
          <Route path="/email-settings" element={<EmailSettings />} />
          <Route path="/settings" element={<PlaceholderPage title="Settings" />} />
        </Route>
      </Routes>
    </Router>
  );
}

export default App;