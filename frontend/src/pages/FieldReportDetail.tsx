import { useEffect, useState } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { Label } from "../components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import { ArrowLeft, Search, FileWarning, Wrench } from "lucide-react";
import { api, type FieldReport } from "../lib/api";
import { toast } from "sonner";

function FieldReportDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [report, setReport] = useState<FieldReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [editData, setEditData] = useState<Partial<FieldReport>>({});

  useEffect(() => {
    if (!id) return;
    const fetchReport = async () => {
      try {
        const data = await api.getFieldReport(id);
        setReport(data);
        setEditData(data);
      } catch (error) {
        toast.error("Failed to load field report");
        console.error("Failed to load field report:", error);
        navigate("/field-reports");
      } finally {
        setLoading(false);
      }
    };
    fetchReport();
  }, [id, navigate]);

  const handleSave = async () => {
    if (!id) return;
    try {
      const updated = await api.updateFieldReport(id, editData);
      setReport(updated);
      setEditing(false);
      toast.success("Field report updated successfully");
    } catch (error) {
      toast.error("Failed to update field report");
      console.error("Failed to update field report:", error);
    }
  };

  const handleStatusChange = async (newStatus: string) => {
    if (!id) return;
    try {
      const updated = await api.updateFieldReport(id, { status: newStatus });
      setReport(updated);
      toast.success(`Status updated to ${newStatus}`);
    } catch (error) {
      toast.error("Failed to update status");
      console.error("Failed to update status:", error);
    }
  };

  const handleCreateNCR = async () => {
    if (!id) return;
    try {
      const ncr = await api.createNCRFromFieldReport(id);
      const updated = await api.getFieldReport(id);
      setReport(updated);
      toast.success(`NCR ${ncr.id} created successfully`);
    } catch (error) {
      toast.error("Failed to create NCR");
      console.error("Failed to create NCR:", error);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
      </div>
    );
  }

  if (!report) return null;

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate("/field-reports")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div className="flex-1">
          <h1 className="text-2xl font-bold">{report.id}: {report.title}</h1>
          <div className="flex gap-2 mt-1">
            <Badge variant="outline">{report.report_type}</Badge>
            <Badge>{report.status}</Badge>
            <Badge>{report.priority}</Badge>
          </div>
        </div>
        <div className="flex gap-2">
          {report.status === "open" && (
            <Button variant="secondary" onClick={() => handleStatusChange("investigating")}>
              <Search className="h-4 w-4 mr-2" />Investigate
            </Button>
          )}
          {(report.status === "open" || report.status === "investigating") && (
            <Button variant="secondary" onClick={() => handleStatusChange("resolved")}>
              <Wrench className="h-4 w-4 mr-2" />Resolve
            </Button>
          )}
          {report.status === "resolved" && (
            <Button variant="secondary" onClick={() => handleStatusChange("closed")}>Close</Button>
          )}
          {!report.ncr_id && (
            <Button variant="destructive" onClick={handleCreateNCR}>
              <FileWarning className="h-4 w-4 mr-2" />Create NCR
            </Button>
          )}
          <Button variant={editing ? "default" : "outline"} onClick={() => editing ? handleSave() : setEditing(true)}>
            {editing ? "Save" : "Edit"}
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <CardHeader><CardTitle>Details</CardTitle></CardHeader>
            <CardContent className="space-y-4">
              {editing ? (
                <>
                  <div>
                    <Label>Title</Label>
                    <Input value={editData.title || ""} onChange={(e) => setEditData({ ...editData, title: e.target.value })} />
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <Label>Report Type</Label>
                      <Select value={editData.report_type || ""} onValueChange={(v) => setEditData({ ...editData, report_type: v })}>
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="failure">Failure</SelectItem>
                          <SelectItem value="complaint">Complaint</SelectItem>
                          <SelectItem value="visit">Site Visit</SelectItem>
                          <SelectItem value="installation">Installation</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div>
                      <Label>Priority</Label>
                      <Select value={editData.priority || ""} onValueChange={(v) => setEditData({ ...editData, priority: v })}>
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="low">Low</SelectItem>
                          <SelectItem value="medium">Medium</SelectItem>
                          <SelectItem value="high">High</SelectItem>
                          <SelectItem value="critical">Critical</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                  <div>
                    <Label>Description</Label>
                    <Textarea value={editData.description || ""} onChange={(e) => setEditData({ ...editData, description: e.target.value })} rows={4} />
                  </div>
                  <div>
                    <Label>Root Cause</Label>
                    <Textarea value={editData.root_cause || ""} onChange={(e) => setEditData({ ...editData, root_cause: e.target.value })} rows={3} />
                  </div>
                  <div>
                    <Label>Resolution</Label>
                    <Textarea value={editData.resolution || ""} onChange={(e) => setEditData({ ...editData, resolution: e.target.value })} rows={3} />
                  </div>
                </>
              ) : (
                <>
                  <div>
                    <Label className="text-muted-foreground text-sm">Description</Label>
                    <p className="mt-1">{report.description || "—"}</p>
                  </div>
                  <div>
                    <Label className="text-muted-foreground text-sm">Root Cause</Label>
                    <p className="mt-1">{report.root_cause || "—"}</p>
                  </div>
                  <div>
                    <Label className="text-muted-foreground text-sm">Resolution</Label>
                    <p className="mt-1">{report.resolution || "—"}</p>
                  </div>
                </>
              )}
            </CardContent>
          </Card>
        </div>

        <div className="space-y-6">
          <Card>
            <CardHeader><CardTitle>Info</CardTitle></CardHeader>
            <CardContent className="space-y-3 text-sm">
              <div><span className="text-muted-foreground">Customer:</span> {report.customer_name || "—"}</div>
              <div><span className="text-muted-foreground">Site:</span> {report.site_location || "—"}</div>
              <div><span className="text-muted-foreground">Device IPN:</span> {report.device_ipn || "—"}</div>
              <div><span className="text-muted-foreground">Serial:</span> {report.device_serial || "—"}</div>
              <div><span className="text-muted-foreground">Reported By:</span> {report.reported_by || "—"}</div>
              <div><span className="text-muted-foreground">Reported At:</span> {report.reported_at?.split(" ")[0] || "—"}</div>
              {report.resolved_at && <div><span className="text-muted-foreground">Resolved At:</span> {report.resolved_at.split(" ")[0]}</div>}
            </CardContent>
          </Card>

          <Card>
            <CardHeader><CardTitle>Links</CardTitle></CardHeader>
            <CardContent className="space-y-3 text-sm">
              {report.ncr_id ? (
                <div>
                  <span className="text-muted-foreground">NCR:</span>{" "}
                  <Link to={`/ncrs/${report.ncr_id}`} className="text-primary underline">{report.ncr_id}</Link>
                </div>
              ) : (
                <div className="text-muted-foreground">No linked NCR</div>
              )}
              {report.eco_id ? (
                <div>
                  <span className="text-muted-foreground">ECO:</span>{" "}
                  <Link to={`/ecos/${report.eco_id}`} className="text-primary underline">{report.eco_id}</Link>
                </div>
              ) : (
                <div className="text-muted-foreground">No linked ECO</div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}

export default FieldReportDetail;
