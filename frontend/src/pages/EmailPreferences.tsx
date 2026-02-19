import { useEffect, useState } from "react";
import { api } from "../lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Checkbox } from "../components/ui/checkbox";
import { Label } from "../components/ui/label";
import { Button } from "../components/ui/button";
import { Mail, Save } from "lucide-react";

const EVENT_LABELS: Record<string, string> = {
  eco_approved: "ECO Approved — notified when your ECO is approved",
  eco_implemented: "ECO Implemented — notified when an ECO is implemented",
  low_stock: "Low Stock Alert — notified when inventory drops below reorder point",
  overdue_work_order: "Overdue Work Order — notified when a work order is past due",
  po_received: "PO Received — notified when your purchase order is received",
  ncr_created: "NCR Created — notified when a new NCR is created",
};

export default function EmailPreferences() {
  const [subs, setSubs] = useState<Record<string, boolean>>({});
  const [original, setOriginal] = useState<Record<string, boolean>>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api.getEmailSubscriptions().then((data) => {
      setSubs(data);
      setOriginal(data);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, []);

  const hasChanges = JSON.stringify(subs) !== JSON.stringify(original);

  const handleSave = async () => {
    setSaving(true);
    try {
      const updated = await api.updateEmailSubscriptions(subs);
      setSubs(updated);
      setOriginal(updated);
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <div className="p-6">Loading...</div>;

  return (
    <div className="p-6 max-w-2xl">
      <h1 className="text-2xl font-bold mb-1 flex items-center gap-2">
        <Mail className="h-6 w-6" /> Email Preferences
      </h1>
      <p className="text-muted-foreground mb-6">Choose which email notifications you want to receive.</p>

      <Card>
        <CardHeader>
          <CardTitle>Notification Types</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {Object.entries(EVENT_LABELS).map(([key, label]) => (
            <div key={key} className="flex items-center space-x-3">
              <Checkbox
                id={key}
                checked={subs[key] ?? true}
                onCheckedChange={(checked) =>
                  setSubs((prev) => ({ ...prev, [key]: !!checked }))
                }
              />
              <Label htmlFor={key} className="cursor-pointer">{label}</Label>
            </div>
          ))}
        </CardContent>
      </Card>

      <Button className="mt-4" onClick={handleSave} disabled={!hasChanges || saving}>
        <Save className="h-4 w-4 mr-2" />
        {saving ? "Saving..." : "Save Preferences"}
      </Button>
    </div>
  );
}
