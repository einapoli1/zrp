import { BrowserRouter as Router, Routes, Route } from "react-router-dom";
import { AppLayout } from "./layouts/AppLayout";
import { Dashboard } from "./pages/Dashboard";
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
          <Route path="/testing" element={<PlaceholderPage title="Testing" />} />
          
          {/* Supply Chain */}
          <Route path="/vendors" element={<PlaceholderPage title="Vendors" />} />
          <Route path="/purchase-orders" element={<PlaceholderPage title="Purchase Orders" />} />
          <Route path="/procurement" element={<PlaceholderPage title="Procurement" />} />
          
          {/* Manufacturing */}
          <Route path="/work-orders" element={<PlaceholderPage title="Work Orders" />} />
          <Route path="/inventory" element={<PlaceholderPage title="Inventory" />} />
          <Route path="/ncrs" element={<PlaceholderPage title="NCRs" />} />
          
          {/* Field & Service */}
          <Route path="/rmas" element={<PlaceholderPage title="RMAs" />} />
          <Route path="/field-reports" element={<PlaceholderPage title="Field Reports" />} />
          
          {/* Sales */}
          <Route path="/quotes" element={<PlaceholderPage title="Quotes" />} />
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