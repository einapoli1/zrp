import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import { Label } from "../components/ui/label";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { ClipboardList, Plus } from "lucide-react";
import { api, type FieldReport } from "../lib/api";
import { toast } from "sonner";
function FieldReports() {
  const navigate = useNavigate();
  const [reports, setReports] = useState<FieldReport[]>([]);
  const [loading, setLoading] = useState(true);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [filterStatus, setFilterStatus] = useState("all");
  const [filterType, setFilterType] = useState("all");
  const [filterPriority, setFilterPriority] = useState("all");
  const [formData, setFormData] = useState({
    title: "",
    report_type: "failure",
    priority: "medium",
    customer_name: "",
    site_location: "",
    device_ipn: "",
    device_serial: "",
    reported_by: "",
    description: "",
    eco_id: "",
  });

  const fetchReports = async () => {
    try {
      const params: Record<string, string> = {};
      if (filterStatus !== "all") params.status = filterStatus;
      if (filterType !== "all") params.report_type = filterType;
      if (filterPriority !== "all") params.priority = filterPriority;
      const data = await api.getFieldReports(Object.keys(params).length > 0 ? params : undefined);
      setReports(data);
    } catch (error) {
      toast.error("Failed to fetch field reports"); console.error("Failed to fetch field reports:", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchReports();
  }, [filterStatus, filterType, filterPriority]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const newReport = await api.createFieldReport(formData);
      setReports([newReport, ...reports]);
      setCreateDialogOpen(false);
      setFormData({
        title: "",
        report_type: "failure",
        priority: "medium",
        customer_name: "",
        site_location: "",
        device_ipn: "",
        device_serial: "",
        reported_by: "",
        description: "",
        eco_id: "",
      });
    } catch (error) {
      toast.error("Failed to create field report"); console.error("Failed to create field report:", error);
    }
  };

  const getPriorityVariant = (priority: string) => {
    switch (priority) {
      case "critical": return "destructive";
      case "high": return "secondary";
      case "medium": return "outline";
      default: return "outline";
    }
  };

  const getStatusVariant = (status: string) => {
    switch (status) {
      case "open": return "destructive";
      case "investigating": return "secondary";
      case "resolved": return "outline";
      case "closed": return "outline";
      default: return "outline";
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading field reports...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Field Reports</h1>
          <p className="text-muted-foreground">
            Track field issues, customer complaints, and site visits
          </p>
        </div>
        <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Create Report
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle>Create Field Report</DialogTitle>
            </DialogHeader>
            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <Label htmlFor="title">Title *</Label>
                <Input
                  id="title"
                  value={formData.title}
                  onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                  placeholder="Brief description of the issue"
                  required
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label>Report Type</Label>
                  <Select value={formData.report_type} onValueChange={(v) => setFormData({ ...formData, report_type: v })}>
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
                  <Select value={formData.priority} onValueChange={(v) => setFormData({ ...formData, priority: v })}>
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
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="customer_name">Customer</Label>
                  <Input id="customer_name" value={formData.customer_name} onChange={(e) => setFormData({ ...formData, customer_name: e.target.value })} />
                </div>
                <div>
                  <Label htmlFor="site_location">Site Location</Label>
                  <Input id="site_location" value={formData.site_location} onChange={(e) => setFormData({ ...formData, site_location: e.target.value })} />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="device_ipn">Device IPN</Label>
                  <Input id="device_ipn" value={formData.device_ipn} onChange={(e) => setFormData({ ...formData, device_ipn: e.target.value })} />
                </div>
                <div>
                  <Label htmlFor="device_serial">Device Serial</Label>
                  <Input id="device_serial" value={formData.device_serial} onChange={(e) => setFormData({ ...formData, device_serial: e.target.value })} />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="reported_by">Reported By</Label>
                  <Input id="reported_by" value={formData.reported_by} onChange={(e) => setFormData({ ...formData, reported_by: e.target.value })} />
                </div>
                <div>
                  <Label htmlFor="eco_id">ECO Reference</Label>
                  <Input id="eco_id" value={formData.eco_id} onChange={(e) => setFormData({ ...formData, eco_id: e.target.value })} placeholder="e.g. ECO-2026-001" />
                </div>
              </div>
              <div>
                <Label htmlFor="description">Description</Label>
                <Textarea id="description" value={formData.description} onChange={(e) => setFormData({ ...formData, description: e.target.value })} rows={4} />
              </div>
              <div className="flex justify-end gap-2">
                <Button type="button" variant="outline" onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
                <Button type="submit">Create</Button>
              </div>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex gap-4 items-end">
            <div>
              <Label>Status</Label>
              <Select value={filterStatus} onValueChange={setFilterStatus}>
                <SelectTrigger className="w-[150px]"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All</SelectItem>
                  <SelectItem value="open">Open</SelectItem>
                  <SelectItem value="investigating">Investigating</SelectItem>
                  <SelectItem value="resolved">Resolved</SelectItem>
                  <SelectItem value="closed">Closed</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>Type</Label>
              <Select value={filterType} onValueChange={setFilterType}>
                <SelectTrigger className="w-[150px]"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All</SelectItem>
                  <SelectItem value="failure">Failure</SelectItem>
                  <SelectItem value="complaint">Complaint</SelectItem>
                  <SelectItem value="visit">Site Visit</SelectItem>
                  <SelectItem value="installation">Installation</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>Priority</Label>
              <Select value={filterPriority} onValueChange={setFilterPriority}>
                <SelectTrigger className="w-[150px]"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All</SelectItem>
                  <SelectItem value="low">Low</SelectItem>
                  <SelectItem value="medium">Medium</SelectItem>
                  <SelectItem value="high">High</SelectItem>
                  <SelectItem value="critical">Critical</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Table */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <ClipboardList className="h-5 w-5" />
            Reports ({reports.length})
          </CardTitle>
        </CardHeader>
        <CardContent>
          {reports.length === 0 ? (
            <p className="text-center text-muted-foreground py-8">No field reports found</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>Title</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Priority</TableHead>
                  <TableHead>Customer</TableHead>
                  <TableHead>Reported</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {reports.map((fr) => (
                  <TableRow key={fr.id} className="cursor-pointer" onClick={() => navigate(`/field-reports/${fr.id}`)}>
                    <TableCell className="font-mono text-sm">{fr.id}</TableCell>
                    <TableCell className="font-medium">{fr.title}</TableCell>
                    <TableCell><Badge variant="outline">{fr.report_type}</Badge></TableCell>
                    <TableCell><Badge variant={getStatusVariant(fr.status)}>{fr.status}</Badge></TableCell>
                    <TableCell><Badge variant={getPriorityVariant(fr.priority)}>{fr.priority}</Badge></TableCell>
                    <TableCell>{fr.customer_name}</TableCell>
                    <TableCell>{fr.reported_at?.split(" ")[0]}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

export default FieldReports;
