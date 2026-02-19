import { useEffect, useState } from "react";
import { toast } from "sonner";
import { api, UndoEntry } from "../lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { RotateCcw } from "lucide-react";

export default function UndoHistory() {
  const [entries, setEntries] = useState<UndoEntry[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchEntries = async () => {
    try {
      const data = await api.getUndoList(50);
      setEntries(data);
    } catch {
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

  if (loading) return <div className="p-6">Loading...</div>;

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Undo History</h1>
        <p className="text-muted-foreground">Recent undoable actions (expires after 24 hours)</p>
      </div>

      {entries.length === 0 ? (
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            No undoable actions available
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {entries.map((entry) => (
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
                    <Button size="sm" variant="outline" onClick={() => handleUndo(entry.id)}>
                      <RotateCcw className="h-3 w-3 mr-1" />
                      Undo
                    </Button>
                  </div>
                </div>
              </CardHeader>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
