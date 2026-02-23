import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Dialog, DialogContent,
  DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { AlertTriangle, Plus } from "lucide-react";
import { toast } from "sonner";
import { api, type NCR } from "../lib/api";
import { EmptyState } from "../components/EmptyState";
import { LoadingState } from "../components/LoadingState";
import { useForm } from "react-hook-form";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "../components/ui/form";

interface CreateNCRData {
  title: string;
  description: string;
  severity: string;
  ipn: string;
}

function NCRs() {
  const navigate = useNavigate();
  const [ncrs, setNCRs] = useState<NCR[]>([]);
  const [loading, setLoading] = useState(true);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);

  const form = useForm<CreateNCRData>({
    defaultValues: {
      title: "",
      description: "",
      severity: "minor",
      ipn: "",
    },
  });

  useEffect(() => {
    const fetchNCRs = async () => {
      try {
        const data = await api.getNCRs();
        setNCRs(data);
      } catch (error: any) {
        toast.error(error.message || "Failed to fetch NCRs");
      } finally {
        setLoading(false);
      }
    };

    fetchNCRs();
  }, []);

  const handleCreateNCR = async (data: CreateNCRData) => {
    try {
      const newNCR = await api.createNCR(data);
      setNCRs([newNCR, ...ncrs]);
      setCreateDialogOpen(false);
      form.reset();
      toast.success("NCR created successfully");
    } catch (error: any) {
      toast.error(error.message || "Failed to create NCR");
    }
  };

  const getSeverityBadgeVariant = (severity: string) => {
    switch (severity) {
      case "critical":
        return "destructive";
      case "major":
        return "secondary";
      case "minor":
      default:
        return "outline";
    }
  };

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case "critical":
        return "bg-red-500";
      case "major":
        return "bg-orange-500";
      case "minor":
      default:
        return "bg-yellow-500";
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Non-Conformance Reports</h1>
          <p className="text-muted-foreground">
            Track quality issues and corrective actions
          </p>
        </div>
        <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Create NCR
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-2xl">
            <Form {...form}>
              <form onSubmit={form.handleSubmit(handleCreateNCR)} className="space-y-6">
                <DialogHeader>
                  <DialogTitle>Create New NCR</DialogTitle>
                  <DialogDescription>
                    Fill out the form below to create a new non-conformance report.
                  </DialogDescription>
                </DialogHeader>
                <div className="space-y-4">
                  <FormField
                    control={form.control}
                    name="title"
                    rules={{ required: 'Title is required' }}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Title *</FormLabel>
                        <FormControl>
                          <Input placeholder="Brief description of the non-conformance" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name="description"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Description</FormLabel>
                        <FormControl>
                          <Textarea placeholder="Detailed description of the issue" rows={3} {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <div className="grid grid-cols-2 gap-4">
                    <FormField
                      control={form.control}
                      name="severity"
                      rules={{ required: 'Severity is required' }}
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Severity *</FormLabel>
                          <Select value={field.value} onValueChange={field.onChange}>
                            <FormControl>
                              <SelectTrigger>
                                <SelectValue />
                              </SelectTrigger>
                            </FormControl>
                            <SelectContent>
                              <SelectItem value="minor">Minor</SelectItem>
                              <SelectItem value="major">Major</SelectItem>
                              <SelectItem value="critical">Critical</SelectItem>
                            </SelectContent>
                          </Select>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="ipn"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Affected IPN</FormLabel>
                          <FormControl>
                            <Input placeholder="Part number affected" {...field} />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>
                </div>
                <div className="flex justify-end gap-2">
                  <Button type="button" variant="outline" onClick={() => setCreateDialogOpen(false)}>
                    Cancel
                  </Button>
                  <Button type="submit">Create NCR</Button>
                </div>
              </form>
            </Form>
          </DialogContent>
        </Dialog>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5" />
            NCR Records
          </CardTitle>
        </CardHeader>
        <CardContent>
          {loading ? (
            <LoadingState variant="table" rows={5} />
          ) : (
            <div className="overflow-x-auto">
              <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>NCR ID</TableHead>
                  <TableHead>Title</TableHead>
                  <TableHead>Severity</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Date</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {ncrs.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} className="p-0">
                      <EmptyState
                        icon={AlertTriangle}
                        title="No NCRs found"
                        description="Create your first Non-Conformance Report to track quality issues"
                        action={
                          <DialogTrigger asChild>
                            <Button>
                              <Plus className="h-4 w-4 mr-2" />
                              Create NCR
                            </Button>
                          </DialogTrigger>
                        }
                      />
                    </TableCell>
                  </TableRow>
                ) : (
                  ncrs.map((ncr) => (
                    <TableRow key={ncr.id} className="cursor-pointer hover:bg-muted/50" onClick={() => navigate(`/ncrs/${ncr.id}`)}>
                      <TableCell className="font-medium">{ncr.id}</TableCell>
                      <TableCell>{ncr.title}</TableCell>
                      <TableCell>
                        <Badge variant={getSeverityBadgeVariant(ncr.severity)} className="flex items-center gap-1 w-fit">
                          <div className={`w-2 h-2 rounded-full ${getSeverityColor(ncr.severity)}`} />
                          {ncr.severity.charAt(0).toUpperCase() + ncr.severity.slice(1)}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant={ncr.status === "closed" || ncr.status === "resolved" ? "default" : "secondary"}>
                          {ncr.status.charAt(0).toUpperCase() + ncr.status.slice(1)}
                        </Badge>
                      </TableCell>
                      <TableCell>{new Date(ncr.created_at).toLocaleDateString()}</TableCell>
                      <TableCell>
                        <Button variant="outline" size="sm" onClick={(e) => { e.stopPropagation(); navigate(`/ncrs/${ncr.id}`); }}>
                          View Details
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
export default NCRs;
