import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { api, type RFQ } from "../lib/api";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Dialog, DialogContent,
  DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { LoadingState } from "../components/LoadingState";
import { EmptyState } from "../components/EmptyState";
import { ErrorState } from "../components/ErrorState";
import { FormField } from "../components/FormField";
import { FileQuestion } from "lucide-react";
import { toast } from "sonner";

const statusColors: Record<string, string> = {
  draft: "secondary",
  sent: "default",
  received: "default",
  awarded: "default",
  closed: "outline",
};

export default function RFQs() {
  const [rfqs, setRfqs] = useState<RFQ[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [newTitle, setNewTitle] = useState("");
  const [newDueDate, setNewDueDate] = useState("");
  const [newNotes, setNewNotes] = useState("");
  const navigate = useNavigate();

  const fetchRFQs = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.getRFQs();
      setRfqs(data);
    } catch (err: any) {
      const message = err?.message || "Failed to load RFQs";
      setError(message);
      toast.error(message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchRFQs();
  }, []);

  const handleCreate = async () => {
    if (!newTitle.trim()) return;
    try {
      const rfq = await api.createRFQ({ title: newTitle, due_date: newDueDate, notes: newNotes });
      setDialogOpen(false);
      setNewTitle("");
      setNewDueDate("");
      setNewNotes("");
      navigate(`/rfqs/${rfq.id}`);
    } catch (err: any) {
      toast.error(err?.message || "Failed to create RFQ");
    }
  };

  if (loading) {
    return <LoadingState variant="spinner" message="Loading RFQs..." />;
  }

  if (error) {
    return (
      <ErrorState
        title="Failed to load RFQs"
        message={error}
        onRetry={fetchRFQs}
      />
    );
  }

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Requests for Quote</h1>
        <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
          <DialogTrigger asChild>
            <Button>Create RFQ</Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>New RFQ</DialogTitle>
            
              <DialogDescription>
                Complete the form below.
              </DialogDescription>
              </DialogHeader>
            <div className="space-y-4">
              <FormField label="Title" htmlFor="rfq-title" required>
                <Input id="rfq-title" value={newTitle} onChange={(e) => setNewTitle(e.target.value)} placeholder="RFQ title" />
              </FormField>
              <FormField label="Due Date" htmlFor="rfq-due">
                <Input id="rfq-due" type="date" value={newDueDate} onChange={(e) => setNewDueDate(e.target.value)} />
              </FormField>
              <FormField label="Notes" htmlFor="rfq-notes">
                <Textarea id="rfq-notes" value={newNotes} onChange={(e) => setNewNotes(e.target.value)} />
              </FormField>
              <Button onClick={handleCreate} disabled={!newTitle.trim()}>Create</Button>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      {rfqs.length === 0 ? (
        <EmptyState
          icon={FileQuestion}
          title="No RFQs found"
          description="Create your first Request for Quote to get started"
          action={
            <Button onClick={() => setDialogOpen(true)}>
              Create RFQ
            </Button>
          }
        />
      ) : (
        <div className="border rounded-lg">
          <table className="w-full">
            <thead>
              <tr className="border-b bg-muted/50">
                <th className="text-left p-3">ID</th>
                <th className="text-left p-3">Title</th>
                <th className="text-left p-3">Status</th>
                <th className="text-left p-3">Due Date</th>
                <th className="text-left p-3">Created</th>
              </tr>
            </thead>
            <tbody>
              {rfqs.map((rfq) => (
                <tr key={rfq.id} className="border-b cursor-pointer hover:bg-muted/30" onClick={() => navigate(`/rfqs/${rfq.id}`)}>
                  <td className="p-3 font-mono text-sm">{rfq.id}</td>
                  <td className="p-3">{rfq.title}</td>
                  <td className="p-3">
                    <Badge variant={statusColors[rfq.status] as any || "secondary"}>{rfq.status}</Badge>
                  </td>
                  <td className="p-3">{rfq.due_date || "—"}</td>
                  <td className="p-3">{rfq.created_at?.split("T")[0]}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
