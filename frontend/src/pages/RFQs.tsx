import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { api, type RFQ } from "../lib/api";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Textarea } from "../components/ui/textarea";

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
  const [dialogOpen, setDialogOpen] = useState(false);
  const [newTitle, setNewTitle] = useState("");
  const [newDueDate, setNewDueDate] = useState("");
  const [newNotes, setNewNotes] = useState("");
  const navigate = useNavigate();

  useEffect(() => {
    api.getRFQs().then((data) => {
      setRfqs(data);
      setLoading(false);
    });
  }, []);

  const handleCreate = async () => {
    if (!newTitle.trim()) return;
    const rfq = await api.createRFQ({ title: newTitle, due_date: newDueDate, notes: newNotes });
    setDialogOpen(false);
    setNewTitle("");
    setNewDueDate("");
    setNewNotes("");
    navigate(`/rfqs/${rfq.id}`);
  };

  if (loading) return <div className="p-6">Loading RFQs...</div>;

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
            </DialogHeader>
            <div className="space-y-4">
              <div>
                <Label htmlFor="rfq-title">Title</Label>
                <Input id="rfq-title" value={newTitle} onChange={(e) => setNewTitle(e.target.value)} placeholder="RFQ title" />
              </div>
              <div>
                <Label htmlFor="rfq-due">Due Date</Label>
                <Input id="rfq-due" type="date" value={newDueDate} onChange={(e) => setNewDueDate(e.target.value)} />
              </div>
              <div>
                <Label htmlFor="rfq-notes">Notes</Label>
                <Textarea id="rfq-notes" value={newNotes} onChange={(e) => setNewNotes(e.target.value)} />
              </div>
              <Button onClick={handleCreate} disabled={!newTitle.trim()}>Create</Button>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      {rfqs.length === 0 ? (
        <p className="text-muted-foreground">No RFQs found</p>
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
