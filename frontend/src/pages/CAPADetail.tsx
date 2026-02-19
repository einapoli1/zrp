import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import { Label } from "../components/ui/label";
import { ArrowLeft, Save, ShieldCheck, CheckCircle } from "lucide-react";
import { api, type CAPA } from "../lib/api";
import { toast } from "sonner";
function CAPADetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [capa, setCAPA] = useState<CAPA | null>(null);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [formData, setFormData] = useState<Partial<CAPA>>({});
  const [error, setError] = useState("");

  useEffect(() => {
    const fetchCAPA = async () => {
      if (!id) return;
      try {
        const data = await api.getCAPA(id);
        setCAPA(data);
        setFormData(data);
      } catch (err) {
        toast.error("Failed to fetch CAPA"); console.error("Failed to fetch CAPA:", error);
      } finally {
        setLoading(false);
      }
    };
    fetchCAPA();
  }, [id]);

  const handleSave = async () => {
    if (!id) return;
    setError("");
    try {
      const updated = await api.updateCAPA(id, formData);
      setCAPA(updated);
      setEditing(false);
    } catch (err: any) {
      const msg = err?.message || "Failed to update CAPA";
      setError(msg);
    }
  };

  const handleApproveQE = async () => {
    if (!id || !capa) return;
    setError("");
    try {
      // Send approval with user identity - backend will validate RBAC and set actual user ID
      const updated = await api.updateCAPA(id, { ...formData, approved_by_qe: "approve" });
      setCAPA(updated);
      setFormData(updated);
      toast.success("QE approval recorded successfully");
    } catch (err: any) {
      const errorMsg = err?.message || "Failed to approve";
      setError(errorMsg);
      toast.error(errorMsg);
    }
  };

  const handleApproveMgr = async () => {
    if (!id || !capa) return;
    setError("");
    try {
      // Send approval with user identity - backend will validate RBAC and set actual user ID
      const updated = await api.updateCAPA(id, { ...formData, approved_by_mgr: "approve" });
      setCAPA(updated);
      setFormData(updated);
      toast.success("Manager approval recorded successfully");
    } catch (err: any) {
      const errorMsg = err?.message || "Failed to approve";
      setError(errorMsg);
      toast.error(errorMsg);
    }
  };

  const statusSteps = ["open", "in-progress", "verification", "closed"];

  if (loading) {
    return <div className="flex items-center justify-center min-h-[400px]"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto" /></div>;
  }

  if (!capa) {
    return <div className="p-6"><p>CAPA not found</p></div>;
  }

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate("/capas")}><ArrowLeft className="h-4 w-4" /></Button>
        <ShieldCheck className="h-5 w-5" />
        <h1 className="text-2xl font-bold">{capa.id}: {capa.title}</h1>
        <Badge variant={capa.type === "corrective" ? "destructive" : "default"}>{capa.type}</Badge>
        <Badge>{capa.status}</Badge>
      </div>

      {error && <div className="bg-red-100 text-red-800 p-3 rounded">{error}</div>}

      {/* Status workflow */}
      <Card>
        <CardHeader><CardTitle className="text-sm">Status Workflow</CardTitle></CardHeader>
        <CardContent>
          <div className="flex items-center gap-2">
            {statusSteps.map((step, i) => (
              <div key={step} className="flex items-center gap-2">
                <div className={`px-3 py-1 rounded text-sm font-medium ${
                  capa.status === step ? "bg-primary text-primary-foreground" :
                  statusSteps.indexOf(capa.status) > i ? "bg-green-100 text-green-800" :
                  "bg-muted text-muted-foreground"
                }`}>{step}</div>
                {i < statusSteps.length - 1 && <span className="text-muted-foreground">→</span>}
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Approval status */}
      <Card>
        <CardHeader><CardTitle className="text-sm">Approval Workflow</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <CheckCircle className={`h-5 w-5 ${capa.approved_by_qe ? "text-green-600" : "text-muted-foreground"}`} />
              <span>QE Approval: {capa.approved_by_qe || "Pending"}</span>
              {capa.approved_by_qe_at && <span className="text-xs text-muted-foreground">({capa.approved_by_qe_at})</span>}
            </div>
            {!capa.approved_by_qe && <Button size="sm" variant="outline" onClick={handleApproveQE}>Approve as QE</Button>}
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <CheckCircle className={`h-5 w-5 ${capa.approved_by_mgr ? "text-green-600" : "text-muted-foreground"}`} />
              <span>Manager Approval: {capa.approved_by_mgr || "Pending"}</span>
              {capa.approved_by_mgr_at && <span className="text-xs text-muted-foreground">({capa.approved_by_mgr_at})</span>}
            </div>
            {!capa.approved_by_mgr && <Button size="sm" variant="outline" onClick={handleApproveMgr}>Approve as Manager</Button>}
          </div>
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle>Details</CardTitle>
            <Button variant="outline" size="sm" onClick={() => setEditing(!editing)}>
              {editing ? "Cancel" : "Edit"}
            </Button>
          </CardHeader>
          <CardContent className="space-y-4">
            <div><Label>Title</Label>
              {editing ? <Input value={formData.title || ""} onChange={(e) => setFormData({ ...formData, title: e.target.value })} />
                : <p className="mt-1">{capa.title}</p>}
            </div>
            <div><Label>Type</Label>
              {editing ? (
                <Select value={formData.type || "corrective"} onValueChange={(v) => setFormData({ ...formData, type: v })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="corrective">Corrective</SelectItem>
                    <SelectItem value="preventive">Preventive</SelectItem>
                  </SelectContent>
                </Select>
              ) : <p className="mt-1">{capa.type}</p>}
            </div>
            <div><Label>Owner</Label>
              {editing ? <Input value={formData.owner || ""} onChange={(e) => setFormData({ ...formData, owner: e.target.value })} />
                : <p className="mt-1">{capa.owner || "—"}</p>}
            </div>
            <div><Label>Due Date</Label>
              {editing ? <Input type="date" value={formData.due_date || ""} onChange={(e) => setFormData({ ...formData, due_date: e.target.value })} />
                : <p className="mt-1">{capa.due_date || "—"}</p>}
            </div>
            <div><Label>Status</Label>
              {editing ? (
                <Select value={formData.status || "open"} onValueChange={(v) => setFormData({ ...formData, status: v })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="open">Open</SelectItem>
                    <SelectItem value="in-progress">In Progress</SelectItem>
                    <SelectItem value="verification">Verification</SelectItem>
                    <SelectItem value="closed">Closed</SelectItem>
                  </SelectContent>
                </Select>
              ) : <p className="mt-1">{capa.status}</p>}
            </div>
            <div><Label>Linked NCR</Label>
              {editing ? <Input value={formData.linked_ncr_id || ""} onChange={(e) => setFormData({ ...formData, linked_ncr_id: e.target.value })} />
                : <p className="mt-1">{capa.linked_ncr_id ? <a href={`/ncrs/${capa.linked_ncr_id}`} className="text-blue-600 underline">{capa.linked_ncr_id}</a> : "—"}</p>}
            </div>
            <div><Label>Linked RMA</Label>
              {editing ? <Input value={formData.linked_rma_id || ""} onChange={(e) => setFormData({ ...formData, linked_rma_id: e.target.value })} />
                : <p className="mt-1">{capa.linked_rma_id ? <a href={`/rmas/${capa.linked_rma_id}`} className="text-blue-600 underline">{capa.linked_rma_id}</a> : "—"}</p>}
            </div>
            {editing && <Button onClick={handleSave}><Save className="mr-2 h-4 w-4" />Save</Button>}
          </CardContent>
        </Card>

        <div className="space-y-6">
          <Card>
            <CardHeader><CardTitle>Root Cause</CardTitle></CardHeader>
            <CardContent>
              {editing ? <Textarea value={formData.root_cause || ""} onChange={(e) => setFormData({ ...formData, root_cause: e.target.value })} rows={4} />
                : <p className="whitespace-pre-wrap">{capa.root_cause || "—"}</p>}
            </CardContent>
          </Card>
          <Card>
            <CardHeader><CardTitle>Action Plan</CardTitle></CardHeader>
            <CardContent>
              {editing ? <Textarea value={formData.action_plan || ""} onChange={(e) => setFormData({ ...formData, action_plan: e.target.value })} rows={4} />
                : <p className="whitespace-pre-wrap">{capa.action_plan || "—"}</p>}
            </CardContent>
          </Card>
          <Card>
            <CardHeader><CardTitle>Effectiveness Verification</CardTitle></CardHeader>
            <CardContent>
              {editing ? <Textarea value={formData.effectiveness_check || ""} onChange={(e) => setFormData({ ...formData, effectiveness_check: e.target.value })} rows={4} placeholder="Document effectiveness verification before closing..." />
                : <p className="whitespace-pre-wrap">{capa.effectiveness_check || "Not yet verified"}</p>}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}

export default CAPADetail;
