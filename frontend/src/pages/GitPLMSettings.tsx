import { useEffect, useState } from "react";
import { api } from "../lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Badge } from "../components/ui/badge";
import { ExternalLink, Save, CheckCircle2, AlertTriangle } from "lucide-react";

function GitPLMSettings() {
  const [baseUrl, setBaseUrl] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<{ ok: boolean; message: string } | null>(null);

  useEffect(() => {
    api.getGitPLMConfig()
      .then((cfg) => setBaseUrl(cfg.base_url || ""))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  const handleSave = async () => {
    setSaving(true);
    try {
      await api.updateGitPLMConfig({ base_url: baseUrl });
      setTestResult({ ok: true, message: "Saved successfully" });
    } catch {
      setTestResult({ ok: false, message: "Failed to save" });
    } finally {
      setSaving(false);
    }
  };

  const handleTestConnection = async () => {
    if (!baseUrl) return;
    setTesting(true);
    setTestResult(null);
    try {
      const resp = await fetch(baseUrl, { mode: "no-cors" });
      setTestResult({ ok: true, message: "Connection successful (reachable)" });
    } catch {
      setTestResult({ ok: false, message: "Could not reach the URL" });
    } finally {
      setTesting(false);
    }
  };

  if (loading) return <div className="p-6">Loading...</div>;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">GitPLM Integration</h1>
        <p className="text-muted-foreground">Configure the connection to your gitplm-ui instance for deep linking.</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center">
            <ExternalLink className="h-5 w-5 mr-2" />
            GitPLM Base URL
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="gitplm-url">Base URL</Label>
            <Input
              id="gitplm-url"
              placeholder="https://gitplm.example.com"
              value={baseUrl}
              onChange={(e) => {
                setBaseUrl(e.target.value);
                setTestResult(null);
              }}
            />
            <p className="text-sm text-muted-foreground">
              The base URL of your gitplm-ui instance. Parts will link to <code>{baseUrl || "https://..."}/parts/IPN</code>
            </p>
          </div>

          <div className="flex items-center space-x-3">
            <Button onClick={handleSave} disabled={saving}>
              <Save className="h-4 w-4 mr-2" />
              {saving ? "Saving..." : "Save"}
            </Button>
            <Button variant="outline" onClick={handleTestConnection} disabled={testing || !baseUrl}>
              {testing ? "Testing..." : "Test Connection"}
            </Button>
          </div>

          {testResult && (
            <div className={`flex items-center space-x-2 text-sm ${testResult.ok ? 'text-green-600' : 'text-red-600'}`}>
              {testResult.ok ? <CheckCircle2 className="h-4 w-4" /> : <AlertTriangle className="h-4 w-4" />}
              <span>{testResult.message}</span>
            </div>
          )}

          <div className="pt-2">
            <Badge variant={baseUrl ? "default" : "secondary"}>
              {baseUrl ? "Configured" : "Not configured"}
            </Badge>
            {baseUrl && (
              <span className="text-sm text-muted-foreground ml-2">
                GitPLM links will appear on parts pages
              </span>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

export default GitPLMSettings;
