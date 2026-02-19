import { useEffect, useState } from "react";
import { toast } from "sonner";
import { api, type UndoEntry, type ChangeEntry } from "../lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { RotateCcw, History } from "lucide-react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import { LoadingState } from "../components/LoadingState";
import { EmptyState } from "../components/EmptyState";
import { ErrorState } from "../components/ErrorState";

export default function UndoHistory() {
  const [undoEntries, setUndoEntries] = useState<UndoEntry[]>([]);
  const [changeEntries, setChangeEntries] = useState<ChangeEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchEntries = async () => {
    setLoading(true);
    setError(null);
    try {
      const [undoData, changeData] = await Promise.all([
        api.getUndoList(50),
        api.getRecentChanges(50),
      ]);
      setUndoEntries(undoData);
      setChangeEntries(changeData);
    } catch (err: any) {
      setError(err.message || "Failed to load undo history");
      toast.error("Failed to load undo history");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchEntries();
  }, []);

  const handleUndo = async (id: number) => {
    try {
      await api.performUndo(id);
      toast.success("Action undone successfully");
      fetchEntries();
    } catch {
      toast.error("Undo failed");
    }
  };

  const handleUndoChange = async (id: number) => {
    try {
      const result = await api.undoChange(id);
      toast(`Undone: ${result.operation} ${result.table_name} ${result.record_id}`, {
        duration: 5000,
        action: {
          label: "Redo",
          onClick: async () => {
            try {
              await api.undoChange(result.redo_id);
              toast.success("Redone successfully");
              fetchEntries();
            } catch {
              toast.error("Redo failed");
            }
          },
        },
      });
      fetchEntries();
    } catch {
      toast.error("Undo failed");
    }
  };

  if (loading) {
    return <LoadingState variant="spinner" message="Loading undo history..." />;
  }

  if (error) {
    return (
      <div className="p-6">
        <ErrorState
          title="Failed to load undo history"
          message={error}
          onRetry={fetchEntries}
        />
      </div>
    );
  }

  return (
    <div className="space-y-6 p-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
          <History className="h-8 w-8" />
          Undo History
        </h1>
        <p className="text-muted-foreground">
          Recent changes and undoable actions. Press Ctrl+Z to undo the last change.
        </p>
      </div>

      <Tabs defaultValue="changes">
        <TabsList className="grid w-full grid-cols-2 md:w-auto md:inline-grid">
          <TabsTrigger value="changes">Change History</TabsTrigger>
          <TabsTrigger value="snapshots">Snapshots (24h)</TabsTrigger>
        </TabsList>

        <TabsContent value="changes" className="space-y-3 mt-4">
          {changeEntries.length === 0 ? (
            <EmptyState
              icon={History}
              title="No changes recorded yet"
              description="Changes you make in the system will appear here and can be undone."
            />
          ) : (
            changeEntries.map((entry) => (
              <Card key={entry.id} className={entry.undone ? "opacity-50" : ""}>
                <CardHeader className="py-3 px-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <CardTitle className="text-sm font-medium">
                        {entry.operation} — {entry.table_name}
                      </CardTitle>
                      <Badge variant="outline">{entry.record_id}</Badge>
                      {entry.undone === 1 && (
                        <Badge variant="secondary">Undone</Badge>
                      )}
                    </div>
                    <div className="flex items-center gap-3">
                      <span className="text-xs text-muted-foreground">
                        {new Date(entry.created_at).toLocaleString()}
                      </span>
                      {!entry.undone && (
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => handleUndoChange(entry.id)}
                        >
                          <RotateCcw className="h-3 w-3 mr-1" />
                          Undo
                        </Button>
                      )}
                    </div>
                  </div>
                </CardHeader>
              </Card>
            ))
          )}
        </TabsContent>

        <TabsContent value="snapshots" className="space-y-3 mt-4">
          {undoEntries.length === 0 ? (
            <Card>
              <CardContent className="py-8 text-center text-muted-foreground">
                No undoable actions available
              </CardContent>
            </Card>
          ) : (
            undoEntries.map((entry) => (
              <Card key={entry.id}>
                <CardHeader className="py-3 px-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <CardTitle className="text-sm font-medium">
                        {entry.action} — {entry.entity_type}
                      </CardTitle>
                      <Badge variant="outline">{entry.entity_id}</Badge>
                    </div>
                    <div className="flex items-center gap-3">
                      <span className="text-xs text-muted-foreground">
                        {new Date(entry.created_at).toLocaleString()}
                      </span>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleUndo(entry.id)}
                      >
                        <RotateCcw className="h-3 w-3 mr-1" />
                        Undo
                      </Button>
                    </div>
                  </div>
                </CardHeader>
              </Card>
            ))
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}
