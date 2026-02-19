import { useEffect, useState } from "react";
import { api } from "../lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Label } from "../components/ui/label";
import {
  Database,
  Download,
  Trash2,
  RotateCcw,
  Plus,
  Loader2,
  AlertTriangle,
} from "lucide-react";
import { LoadingState } from "../components/LoadingState";
import { EmptyState } from "../components/EmptyState";
import { ErrorState } from "../components/ErrorState";

interface BackupInfo {
  filename: string;
  size: number;
  created_at: string;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
}

function Backups() {
  const [backups, setBackups] = useState<BackupInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [restoring, setRestoring] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const fetchBackups = async () => {
    try {
      const resp = await api.getBackups();
      setBackups(resp);
      setError(null);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchBackups();
  }, []);

  const handleCreate = async () => {
    setCreating(true);
    try {
      await api.createBackup();
      await fetchBackups();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (filename: string) => {
    if (!confirm(`Delete backup ${filename}?`)) return;
    try {
      await api.deleteBackup(filename);
      await fetchBackups();
    } catch (e: any) {
      setError(e.message);
    }
  };

  const handleDownload = (filename: string) => {
    window.open(`/api/v1/admin/backups/${filename}`, "_blank");
  };

  const handleRestore = async (filename: string) => {
    if (
      !confirm(
        `⚠️ DANGER: Restore database from ${filename}?\n\nThis will replace the current database. A pre-restore backup will be created automatically.`
      )
    )
      return;
    setRestoring(filename);
    try {
      await api.restoreBackup(filename);
      alert("Database restored successfully. The page will reload.");
      window.location.reload();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setRestoring(null);
    }
  };

  if (loading) {
    return <LoadingState variant="spinner" message="Loading backups..." />;
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Backups</h1>
          <p className="text-muted-foreground">
            Manage database backups. Auto-backups run daily at 2:00 AM with 7-day
            retention.
          </p>
        </div>
        <Button onClick={handleCreate} disabled={creating}>
          {creating ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <Plus className="mr-2 h-4 w-4" />
          )}
          Create Backup
        </Button>
      </div>

      {error && (
        <ErrorState
          variant="inline"
          title="Backup operation failed"
          message={error}
          onRetry={fetchBackups}
        />
      )}

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Database className="h-5 w-5" />
            Available Backups
            <Badge variant="secondary">{backups.length}</Badge>
          </CardTitle>
        </CardHeader>
        <CardContent>
          {backups.length === 0 ? (
            <EmptyState
              icon={Database}
              title="No backups yet"
              description="Create your first backup to protect your database. Auto-backups will also run daily at 2:00 AM."
              action={
                <Button onClick={handleCreate} disabled={creating}>
                  <Plus className="mr-2 h-4 w-4" />
                  Create First Backup
                </Button>
              }
            />
          ) : (
            <div className="divide-y">
              {backups.map((backup) => (
                <div
                  key={backup.filename}
                  className="flex flex-col sm:flex-row sm:items-center sm:justify-between py-3 gap-2"
                >
                  <div className="flex-1 min-w-0">
                    <p className="font-mono text-sm truncate">{backup.filename}</p>
                    <p className="text-xs text-muted-foreground">
                      {formatBytes(backup.size)} •{" "}
                      {new Date(backup.created_at).toLocaleString()}
                    </p>
                  </div>
                  <div className="flex gap-2 flex-shrink-0">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleDownload(backup.filename)}
                      aria-label={`Download ${backup.filename}`}
                      title="Download backup"
                    >
                      <Download className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleRestore(backup.filename)}
                      disabled={restoring === backup.filename}
                      aria-label={`Restore ${backup.filename}`}
                      title="Restore from this backup"
                    >
                      {restoring === backup.filename ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <RotateCcw className="h-4 w-4" />
                      )}
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleDelete(backup.filename)}
                      className="text-destructive hover:text-destructive"
                      aria-label={`Delete ${backup.filename}`}
                      title="Delete this backup"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

export default Backups;
