import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Store, Save, ExternalLink } from "lucide-react";
import { api } from "../lib/api";
import { LoadingState } from "../components/LoadingState";
import { ErrorState } from "../components/ErrorState";
import { FormField } from "../components/FormField";
import { toast } from "sonner";

export default function DistributorSettings() {
  const [digikeyClientId, setDigikeyClientId] = useState("");
  const [digikeyClientSecret, setDigikeyClientSecret] = useState("");
  const [mouserKey, setMouserKey] = useState("");
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    setLoading(true);
    setError(null);
    try {
      const settings = await api.getDistributorSettings();
      setDigikeyClientId(settings.digikey?.client_id || "");
      setDigikeyClientSecret(settings.digikey?.client_secret || "");
      setMouserKey(settings.mouser?.api_key || "");
    } catch (err: any) {
      const message = err?.message || "Failed to load settings";
      setError(message);
      toast.error(message);
    } finally {
      setLoading(false);
    }
  };

  const saveDigikey = async () => {
    setSaving(true);
    try {
      await api.updateDigikeySettings({ client_id: digikeyClientId, client_secret: digikeyClientSecret });
      toast.success("Digikey settings saved");
    } catch (err: any) {
      toast.error(err?.message || "Failed to save Digikey settings");
    } finally {
      setSaving(false);
    }
  };

  const saveMouser = async () => {
    setSaving(true);
    try {
      await api.updateMouserSettings({ api_key: mouserKey });
      toast.success("Mouser settings saved");
    } catch (err: any) {
      toast.error(err?.message || "Failed to save Mouser settings");
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return <LoadingState variant="spinner" message="Loading distributor settings..." />;
  }

  if (error) {
    return (
      <ErrorState
        title="Failed to load settings"
        message={error}
        onRetry={loadSettings}
      />
    );
  }

  return (
    <div className="space-y-6 p-6 max-w-2xl">
      <div className="flex items-center gap-2">
        <Store className="h-6 w-6" />
        <h1 className="text-2xl font-bold">Distributor API Settings</h1>
      </div>

      <p className="text-sm text-muted-foreground">
        Configure API credentials to fetch live pricing and stock data from distributors.
        Keys are stored securely in the database.
      </p>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Digikey</CardTitle>
            <a href="https://developer.digikey.com/" target="_blank" rel="noopener noreferrer"
               className="text-xs text-blue-600 hover:underline flex items-center gap-1">
              <ExternalLink className="h-3 w-3" /> Get API credentials
            </a>
          </div>
          <p className="text-sm text-muted-foreground">
            Uses OAuth2 Client Credentials flow with the Product Search v4 API.
            Create an app at the Digikey Developer Portal to get your Client ID and Secret.
          </p>
        </CardHeader>
        <CardContent className="space-y-4">
          <FormField label="Client ID" htmlFor="digikey-client-id">
            <Input
              id="digikey-client-id"
              value={digikeyClientId}
              onChange={e => setDigikeyClientId(e.target.value)}
              placeholder="Digikey OAuth2 Client ID"
            />
          </FormField>
          <FormField label="Client Secret" htmlFor="digikey-secret">
            <Input
              id="digikey-secret"
              value={digikeyClientSecret}
              onChange={e => setDigikeyClientSecret(e.target.value)}
              placeholder="Digikey OAuth2 Client Secret"
              type="password"
            />
          </FormField>
          <Button onClick={saveDigikey} disabled={saving}>
            <Save className="h-4 w-4 mr-1" /> Save Digikey Settings
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Mouser</CardTitle>
            <a href="https://www.mouser.com/api-hub/" target="_blank" rel="noopener noreferrer"
               className="text-xs text-blue-600 hover:underline flex items-center gap-1">
              <ExternalLink className="h-3 w-3" /> Get API key
            </a>
          </div>
          <p className="text-sm text-muted-foreground">
            Uses the Mouser Search API v2. Register at the Mouser API Hub for a free API key.
          </p>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            Get your API key from <a href="https://api.mouser.com/api/docs/" target="_blank" rel="noopener noreferrer" className="underline">api.mouser.com</a> (Search API v2).
          </p>
          <FormField label="API Key" htmlFor="mouser-key">
            <Input
              id="mouser-key"
              value={mouserKey}
              onChange={e => setMouserKey(e.target.value)}
              placeholder="Mouser Search API Key"
              type="password"
            />
          </FormField>
          <Button onClick={saveMouser} disabled={saving}>
            <Save className="h-4 w-4 mr-1" /> Save Mouser Settings
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
