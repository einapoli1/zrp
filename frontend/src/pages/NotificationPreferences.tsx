import { useEffect, useState } from "react";
import { api } from "../lib/api";
import type { NotificationTypeInfo } from "../lib/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import { toast } from "sonner";
import { Bell, Save, RotateCcw, Package, Clock, AlertTriangle, CheckCircle, CheckSquare, Truck, Check, AlertCircle } from "lucide-react";

const ICON_MAP: Record<string, React.ComponentType<any>> = {
  package: Package,
  clock: Clock,
  "alert-triangle": AlertTriangle,
  "check-circle": CheckCircle,
  "check-square": CheckSquare,
  truck: Truck,
  check: Check,
  "alert-circle": AlertCircle,
};

interface PrefState {
  enabled: boolean;
  delivery_method: string;
  threshold_value: number | null;
}

export default function NotificationPreferences() {
  const [types, setTypes] = useState<NotificationTypeInfo[]>([]);
  const [prefs, setPrefs] = useState<Record<string, PrefState>>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    Promise.all([api.getNotificationTypes(), api.getNotificationPreferences()])
      .then(([typesData, prefsData]) => {
        setTypes(typesData);
        const prefMap: Record<string, PrefState> = {};
        for (const p of prefsData) {
          prefMap[p.notification_type] = {
            enabled: p.enabled,
            delivery_method: p.delivery_method,
            threshold_value: p.threshold_value,
          };
        }
        // Fill in defaults for any missing types
        for (const t of typesData) {
          if (!prefMap[t.type]) {
            prefMap[t.type] = {
              enabled: true,
              delivery_method: "in_app",
              threshold_value: t.threshold_default ?? null,
            };
          }
        }
        setPrefs(prefMap);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  const handleSave = async () => {
    setSaving(true);
    try {
      const payload = types.map((t) => ({
        notification_type: t.type,
        enabled: prefs[t.type]?.enabled ?? true,
        delivery_method: prefs[t.type]?.delivery_method ?? "in_app",
        threshold_value: prefs[t.type]?.threshold_value ?? null,
      }));
      await api.updateNotificationPreferences(payload);
      toast.success("Notification preferences saved");
    } catch (err: any) {
      toast.error("Failed to save: " + (err.message || "Unknown error"));
    } finally {
      setSaving(false);
    }
  };

  const handleReset = () => {
    const prefMap: Record<string, PrefState> = {};
    for (const t of types) {
      prefMap[t.type] = {
        enabled: true,
        delivery_method: "in_app",
        threshold_value: t.threshold_default ?? null,
      };
    }
    setPrefs(prefMap);
    toast.info("Reset to defaults — click Save to apply");
  };

  const updatePref = (type: string, field: keyof PrefState, value: any) => {
    setPrefs((prev) => ({
      ...prev,
      [type]: { ...prev[type], [field]: value },
    }));
  };

  if (loading) return <div className="text-muted-foreground p-6">Loading...</div>;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Bell className="h-6 w-6" />
            Notification Preferences
          </h1>
          <p className="text-muted-foreground mt-1">
            Configure which notifications you receive and how they're delivered.
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={handleReset}>
            <RotateCcw className="h-4 w-4 mr-1" />
            Reset to Defaults
          </Button>
          <Button onClick={handleSave} disabled={saving}>
            <Save className="h-4 w-4 mr-1" />
            {saving ? "Saving..." : "Save Preferences"}
          </Button>
        </div>
      </div>

      <div className="grid gap-4">
        {types.map((t) => {
          const pref = prefs[t.type] || { enabled: true, delivery_method: "in_app", threshold_value: null };
          const IconComp = ICON_MAP[t.icon] || Bell;

          return (
            <Card key={t.type} className={!pref.enabled ? "opacity-60" : ""}>
              <CardHeader className="pb-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <IconComp className="h-5 w-5 text-muted-foreground" />
                    <div>
                      <CardTitle className="text-base">{t.name}</CardTitle>
                      <CardDescription>{t.description}</CardDescription>
                    </div>
                  </div>
                  <label className="flex items-center gap-2 cursor-pointer">
                    <span className="text-sm text-muted-foreground">
                      {pref.enabled ? "Enabled" : "Disabled"}
                    </span>
                    <input
                      type="checkbox"
                      checked={pref.enabled}
                      onChange={(e) => updatePref(t.type, "enabled", e.target.checked)}
                      className="h-5 w-5 rounded"
                    />
                  </label>
                </div>
              </CardHeader>
              {pref.enabled && (
                <CardContent className="pt-0">
                  <div className="flex flex-wrap items-center gap-4">
                    <div className="space-y-1">
                      <Label className="text-xs text-muted-foreground">Delivery Method</Label>
                      <Select
                        value={pref.delivery_method}
                        onValueChange={(v) => updatePref(t.type, "delivery_method", v)}
                      >
                        <SelectTrigger className="w-[140px]">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="in_app">In-App</SelectItem>
                          <SelectItem value="email">Email</SelectItem>
                          <SelectItem value="both">Both</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    {t.has_threshold && (
                      <div className="space-y-1">
                        <Label className="text-xs text-muted-foreground">
                          {t.threshold_label || "Threshold"}
                        </Label>
                        <Input
                          type="number"
                          className="w-[120px]"
                          value={pref.threshold_value ?? ""}
                          onChange={(e) =>
                            updatePref(
                              t.type,
                              "threshold_value",
                              e.target.value === "" ? null : Number(e.target.value)
                            )
                          }
                        />
                      </div>
                    )}
                  </div>
                </CardContent>
              )}
            </Card>
          );
        })}
      </div>
    </div>
  );
}
