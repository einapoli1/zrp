import { useEffect, useState } from "react";
import { api } from "../lib/api";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import { toast } from "sonner";
import { Settings as SettingsIcon, Mail, Globe, Database, Users, Wrench } from "lucide-react";
import EmailSettings from "./EmailSettings";
import GitPLMSettings from "./GitPLMSettings";
import GitDocsSettings from "./GitDocsSettings";
import DistributorSettings from "./DistributorSettings";
import Backups from "./Backups";

interface GeneralSettings {
  app_name: string;
  company_name: string;
  company_address: string;
  currency: string;
  date_format: string;
}

function GeneralSettingsTab() {
  const [settings, setSettings] = useState<GeneralSettings>({
    app_name: "ZRP",
    company_name: "",
    company_address: "",
    currency: "USD",
    date_format: "YYYY-MM-DD",
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api.getGeneralSettings().then((s) => {
      setSettings(s);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, []);

  const handleSave = async () => {
    setSaving(true);
    try {
      await api.updateGeneralSettings(settings);
      toast.success("General settings saved");
    } catch (err: any) {
      toast.error("Failed to save settings: " + (err.message || "Unknown error"));
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <div className="text-muted-foreground">Loading...</div>;

  return (
    <Card>
      <CardHeader>
        <CardTitle>General Settings</CardTitle>
        <CardDescription>Configure basic application settings</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="app_name">App Name</Label>
          <Input
            id="app_name"
            value={settings.app_name}
            onChange={(e) => setSettings({ ...settings, app_name: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="company_name">Company Name</Label>
          <Input
            id="company_name"
            value={settings.company_name}
            onChange={(e) => setSettings({ ...settings, company_name: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="company_address">Company Address</Label>
          <Input
            id="company_address"
            value={settings.company_address}
            onChange={(e) => setSettings({ ...settings, company_address: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="currency">Currency</Label>
          <Select value={settings.currency} onValueChange={(v) => setSettings({ ...settings, currency: v })}>
            <SelectTrigger id="currency">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="USD">USD — US Dollar</SelectItem>
              <SelectItem value="EUR">EUR — Euro</SelectItem>
              <SelectItem value="GBP">GBP — British Pound</SelectItem>
              <SelectItem value="JPY">JPY — Japanese Yen</SelectItem>
              <SelectItem value="CAD">CAD — Canadian Dollar</SelectItem>
              <SelectItem value="AUD">AUD — Australian Dollar</SelectItem>
              <SelectItem value="CHF">CHF — Swiss Franc</SelectItem>
              <SelectItem value="CNY">CNY — Chinese Yuan</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-2">
          <Label htmlFor="date_format">Date Format</Label>
          <Select value={settings.date_format} onValueChange={(v) => setSettings({ ...settings, date_format: v })}>
            <SelectTrigger id="date_format">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="YYYY-MM-DD">YYYY-MM-DD</SelectItem>
              <SelectItem value="MM/DD/YYYY">MM/DD/YYYY</SelectItem>
              <SelectItem value="DD/MM/YYYY">DD/MM/YYYY</SelectItem>
              <SelectItem value="DD.MM.YYYY">DD.MM.YYYY</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button onClick={handleSave} disabled={saving}>
          {saving ? "Saving..." : "Save Settings"}
        </Button>
      </CardContent>
    </Card>
  );
}

function UsersAuthTab() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Users & Authentication</CardTitle>
        <CardDescription>Manage user accounts and authentication settings</CardDescription>
      </CardHeader>
      <CardContent>
        <p className="text-muted-foreground">
          User management is available at the dedicated{" "}
          <a href="/users" className="text-primary underline">Users</a>{" "}
          and{" "}
          <a href="/api-keys" className="text-primary underline">API Keys</a>{" "}
          pages.
        </p>
      </CardContent>
    </Card>
  );
}

export default function Settings() {
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <SettingsIcon className="h-8 w-8" />
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Settings</h1>
          <p className="text-muted-foreground">Manage application configuration</p>
        </div>
      </div>

      <Tabs defaultValue="general" className="space-y-4">
        <TabsList>
          <TabsTrigger value="general"><Wrench className="h-4 w-4 mr-1" />General</TabsTrigger>
          <TabsTrigger value="email"><Mail className="h-4 w-4 mr-1" />Email / SMTP</TabsTrigger>
          <TabsTrigger value="distributors"><Globe className="h-4 w-4 mr-1" />Distributor APIs</TabsTrigger>
          <TabsTrigger value="gitplm">GitPLM</TabsTrigger>
          <TabsTrigger value="git-docs">Git Docs</TabsTrigger>
          <TabsTrigger value="backups"><Database className="h-4 w-4 mr-1" />Backups</TabsTrigger>
          <TabsTrigger value="users"><Users className="h-4 w-4 mr-1" />Users / Auth</TabsTrigger>
        </TabsList>

        <TabsContent value="general">
          <GeneralSettingsTab />
        </TabsContent>

        <TabsContent value="email">
          <EmailSettings />
        </TabsContent>

        <TabsContent value="distributors">
          <DistributorSettings />
        </TabsContent>

        <TabsContent value="gitplm">
          <GitPLMSettings />
        </TabsContent>

        <TabsContent value="git-docs">
          <GitDocsSettings />
        </TabsContent>

        <TabsContent value="backups">
          <Backups />
        </TabsContent>

        <TabsContent value="users">
          <UsersAuthTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}
