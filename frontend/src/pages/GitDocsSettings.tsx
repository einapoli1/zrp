import { useEffect, useState } from "react";
import { api, type GitDocsConfig } from "../lib/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { toast } from "sonner";

export default function GitDocsSettings() {
  const [config, setConfig] = useState<GitDocsConfig>({
    repo_url: "",
    branch: "main",
    token: "",
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api.getGitDocsSettings().then((cfg) => {
      setConfig(cfg);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, []);

  const handleSave = async () => {
    setSaving(true);
    try {
      await api.updateGitDocsSettings(config);
      toast.success("Git docs settings saved");
    } catch (err: any) {
      toast.error("Failed to save: " + (err.message || "Unknown error"));
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <div className="text-muted-foreground">Loading...</div>;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Git Document Repository</CardTitle>
        <CardDescription>
          Connect a Git repository to sync and version-control documents externally
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="git_repo_url">Repository URL</Label>
          <Input
            id="git_repo_url"
            placeholder="https://github.com/org/docs-repo.git"
            value={config.repo_url}
            onChange={(e) => setConfig({ ...config, repo_url: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="git_branch">Branch</Label>
          <Input
            id="git_branch"
            placeholder="main"
            value={config.branch}
            onChange={(e) => setConfig({ ...config, branch: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="git_token">Access Token</Label>
          <Input
            id="git_token"
            type="password"
            placeholder="Enter token (leave blank to keep existing)"
            value={config.token}
            onChange={(e) => setConfig({ ...config, token: e.target.value })}
          />
          <p className="text-xs text-muted-foreground">
            Used for authentication when pushing/pulling from the repository
          </p>
        </div>
        <Button onClick={handleSave} disabled={saving}>
          {saving ? "Saving..." : "Save Git Docs Settings"}
        </Button>
      </CardContent>
    </Card>
  );
}
