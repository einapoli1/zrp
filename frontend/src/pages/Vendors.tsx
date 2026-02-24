import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { Checkbox } from "../components/ui/checkbox";
import { Label } from "../components/ui/label";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "../components/ui/alert-dialog";
import { Factory, Plus, Edit, Trash2, Search, CheckCircle, XCircle } from "lucide-react";
import { api, type Manufacturer } from "../lib/api";
import { toast } from "sonner";
import { Breadcrumb } from "../components/ui/breadcrumb";
import { LoadingState } from "../components/LoadingState";

export function Vendors() {
  const [manufacturers, setManufacturers] = useState<Manufacturer[]>([]);
  const [filteredManufacturers, setFilteredManufacturers] = useState<Manufacturer[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState("");
  
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingManufacturer, setEditingManufacturer] = useState<Manufacturer | null>(null);
  const [manufacturerForm, setManufacturerForm] = useState({
    name: "",
    contact_name: "",
    contact_email: "",
    contact_phone: "",
    website: "",
    notes: "",
    approved: true,
  });
  const [formErrors, setFormErrors] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [manufacturerToDelete, setManufacturerToDelete] = useState<Manufacturer | null>(null);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    fetchManufacturers();
  }, []);

  useEffect(() => {
    filterManufacturers();
  }, [searchQuery, manufacturers]);

  const fetchManufacturers = async () => {
    setLoading(true);
    try {
      const data = await api.getManufacturers();
      console.log("Manufacturers fetched:", data);
      console.log("Raw manufacturers data:", data, "isArray:", Array.isArray(data));
      setManufacturers(Array.isArray(data) ? data : []);
    } catch (error: any) {
      console.error("Failed to fetch manufacturers:", error);
      toast.error(error.message || "Failed to fetch manufacturers");
    } finally {
      setLoading(false);
    }
  };

  const filterManufacturers = () => {
    if (!searchQuery.trim()) {
      setFilteredManufacturers(manufacturers);
      return;
    }

    const query = searchQuery.toLowerCase();
    const filtered = manufacturers.filter(
      (m) =>
        m.name.toLowerCase().includes(query) ||
        m.contact_name?.toLowerCase().includes(query) ||
        m.contact_email?.toLowerCase().includes(query)
    );
    setFilteredManufacturers(filtered);
  };

  const openAddDialog = () => {
    setEditingManufacturer(null);
    setManufacturerForm({
      name: "",
      contact_name: "",
      contact_email: "",
      contact_phone: "",
      website: "",
      notes: "",
      approved: true,
    });
    setFormErrors({});
    setDialogOpen(true);
  };

  const openEditDialog = (manufacturer: Manufacturer) => {
    setEditingManufacturer(manufacturer);
    setManufacturerForm({
      name: manufacturer.name,
      contact_name: manufacturer.contact_name || "",
      contact_email: manufacturer.contact_email || "",
      contact_phone: manufacturer.contact_phone || "",
      website: manufacturer.website || "",
      notes: manufacturer.notes || "",
      approved: manufacturer.approved,
    });
    setFormErrors({});
    setDialogOpen(true);
  };

  const validateForm = (): boolean => {
    const errors: Record<string, string> = {};

    if (!manufacturerForm.name.trim()) {
      errors.name = "Name is required";
    }

    if (manufacturerForm.contact_email && !isValidEmail(manufacturerForm.contact_email)) {
      errors.contact_email = "Invalid email address";
    }

    if (manufacturerForm.website && !isValidUrl(manufacturerForm.website)) {
      errors.website = "Invalid URL";
    }

    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const isValidEmail = (email: string): boolean => {
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
  };

  const isValidUrl = (url: string): boolean => {
    try {
      new URL(url);
      return true;
    } catch {
      return false;
    }
  };

  const handleSave = async () => {
    if (!validateForm()) {
      return;
    }

    setSaving(true);
    try {
      if (editingManufacturer) {
        await api.updateManufacturer(editingManufacturer.id, manufacturerForm);
        toast.success("Manufacturer updated successfully");
      } else {
        await api.createManufacturer(manufacturerForm);
        toast.success("Manufacturer added successfully");
      }
      
      setDialogOpen(false);
      await fetchManufacturers();
    } catch (error: any) {
      console.error("Failed to save manufacturer:", error);
      const errorMsg = error.message || "Failed to save manufacturer";
      
      // Check for fuzzy match suggestion in error message
      if (errorMsg.includes("similar") || errorMsg.includes("Did you mean")) {
        toast.error(errorMsg, { duration: 6000 });
      } else {
        toast.error(errorMsg);
      }
    } finally {
      setSaving(false);
    }
  };

  const openDeleteDialog = (manufacturer: Manufacturer) => {
    setManufacturerToDelete(manufacturer);
    setDeleteDialogOpen(true);
  };

  const handleDelete = async () => {
    if (!manufacturerToDelete) return;

    setDeleting(true);
    try {
      await api.deleteManufacturer(manufacturerToDelete.id);
      toast.success("Manufacturer deleted successfully");
      setDeleteDialogOpen(false);
      await fetchManufacturers();
    } catch (error: any) {
      console.error("Failed to delete manufacturer:", error);
      const errorMsg = error.message || "Failed to delete manufacturer";
      
      // Check for 409 conflict (parts reference this manufacturer)
      if (error.message?.includes("409") || errorMsg.includes("reference") || errorMsg.includes("parts")) {
        toast.error("Cannot delete: This manufacturer is referenced by existing parts", {
          duration: 5000,
        });
      } else {
        toast.error(errorMsg);
      }
    } finally {
      setDeleting(false);
    }
  };

  if (loading) {
    return <LoadingState />;
  }

  return (
    <div className="container mx-auto py-6 space-y-6">
      <Breadcrumb
        items={[
          { label: "Home", href: "/" },
          { label: "Manufacturers", href: "/vendors" },
        ]}
      />

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold flex items-center">
            <Factory className="h-8 w-8 mr-3" />
            Manufacturers
          </h1>
          <p className="text-muted-foreground mt-1">
            Manage manufacturers and suppliers for parts
          </p>
        </div>
        <Button onClick={openAddDialog} data-testid="add-manufacturer-btn">
          <Plus className="h-4 w-4 mr-2" />
          Add Manufacturer
        </Button>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>
              All Manufacturers
              {manufacturers.length > 0 && (
                <Badge variant="secondary" className="ml-2">{manufacturers.length}</Badge>
              )}
            </CardTitle>
            <div className="flex items-center space-x-2">
              <div className="relative w-64">
                <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="Search manufacturers..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-8"
                  data-testid="search-input"
                />
              </div>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {filteredManufacturers.length > 0 ? (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Contact</TableHead>
                    <TableHead>Email</TableHead>
                    <TableHead>Phone</TableHead>
                    <TableHead>Website</TableHead>
                    <TableHead>Approved</TableHead>
                    <TableHead className="w-[100px]">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredManufacturers.map((mfg) => (
                    <TableRow key={mfg.id} data-testid={`manufacturer-row-${mfg.id}`}>
                      <TableCell className="font-medium">{mfg.name}</TableCell>
                      <TableCell>{mfg.contact_name || "—"}</TableCell>
                      <TableCell>{mfg.contact_email || "—"}</TableCell>
                      <TableCell>{mfg.contact_phone || "—"}</TableCell>
                      <TableCell>
                        {mfg.website ? (
                          <a
                            href={mfg.website}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-primary hover:underline"
                          >
                            {new URL(mfg.website).hostname}
                          </a>
                        ) : (
                          "—"
                        )}
                      </TableCell>
                      <TableCell>
                        {mfg.approved ? (
                          <Badge variant="default" className="bg-green-600" data-testid={`approved-badge-${mfg.id}`}>
                            <CheckCircle className="h-3 w-3 mr-1" />
                            Approved
                          </Badge>
                        ) : (
                          <Badge variant="secondary" data-testid={`unapproved-badge-${mfg.id}`}>
                            <XCircle className="h-3 w-3 mr-1" />
                            Pending
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center space-x-2">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => openEditDialog(mfg)}
                            data-testid={`edit-manufacturer-${mfg.id}`}
                          >
                            <Edit className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => openDeleteDialog(mfg)}
                            data-testid={`delete-manufacturer-${mfg.id}`}
                          >
                            <Trash2 className="h-4 w-4 text-destructive" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ) : manufacturers.length === 0 ? (
            <div className="text-center py-12 text-muted-foreground">
              <Factory className="h-12 w-12 mx-auto mb-3 opacity-50" />
              <p className="text-lg font-medium">No manufacturers added</p>
              <p className="text-sm mt-1">Get started by adding your first manufacturer</p>
              <Button variant="outline" size="sm" className="mt-4" onClick={openAddDialog}>
                <Plus className="h-4 w-4 mr-2" />
                Add Manufacturer
              </Button>
            </div>
          ) : (
            <div className="text-center py-12 text-muted-foreground">
              <Search className="h-12 w-12 mx-auto mb-3 opacity-50" />
              <p className="text-lg font-medium">No results found</p>
              <p className="text-sm mt-1">Try adjusting your search query</p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Add/Edit Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-2xl" data-testid="manufacturer-dialog">
          <DialogHeader>
            <DialogTitle>
              {editingManufacturer ? "Edit Manufacturer" : "Add Manufacturer"}
            </DialogTitle>
            <DialogDescription>
              {editingManufacturer
                ? "Update manufacturer information."
                : "Add a new manufacturer to the system."}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="name">Name *</Label>
              <Input
                id="name"
                value={manufacturerForm.name}
                onChange={(e) =>
                  setManufacturerForm({ ...manufacturerForm, name: e.target.value })
                }
                placeholder="e.g., Texas Instruments"
                data-testid="name-input"
              />
              {formErrors.name && (
                <p className="text-sm text-destructive">{formErrors.name}</p>
              )}
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="contact_name">Contact Name</Label>
                <Input
                  id="contact_name"
                  value={manufacturerForm.contact_name}
                  onChange={(e) =>
                    setManufacturerForm({ ...manufacturerForm, contact_name: e.target.value })
                  }
                  placeholder="e.g., John Doe"
                  data-testid="contact-name-input"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="contact_email">Contact Email</Label>
                <Input
                  id="contact_email"
                  type="email"
                  value={manufacturerForm.contact_email}
                  onChange={(e) =>
                    setManufacturerForm({ ...manufacturerForm, contact_email: e.target.value })
                  }
                  placeholder="e.g., sales@example.com"
                  data-testid="contact-email-input"
                />
                {formErrors.contact_email && (
                  <p className="text-sm text-destructive">{formErrors.contact_email}</p>
                )}
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="contact_phone">Contact Phone</Label>
                <Input
                  id="contact_phone"
                  value={manufacturerForm.contact_phone}
                  onChange={(e) =>
                    setManufacturerForm({ ...manufacturerForm, contact_phone: e.target.value })
                  }
                  placeholder="e.g., +1-555-123-4567"
                  data-testid="contact-phone-input"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="website">Website</Label>
                <Input
                  id="website"
                  value={manufacturerForm.website}
                  onChange={(e) =>
                    setManufacturerForm({ ...manufacturerForm, website: e.target.value })
                  }
                  placeholder="e.g., https://example.com"
                  data-testid="website-input"
                />
                {formErrors.website && (
                  <p className="text-sm text-destructive">{formErrors.website}</p>
                )}
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="notes">Notes</Label>
              <Textarea
                id="notes"
                value={manufacturerForm.notes}
                onChange={(e) =>
                  setManufacturerForm({ ...manufacturerForm, notes: e.target.value })
                }
                placeholder="Additional notes..."
                rows={3}
                data-testid="notes-input"
              />
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="approved"
                checked={manufacturerForm.approved}
                onCheckedChange={(checked) =>
                  setManufacturerForm({ ...manufacturerForm, approved: checked === true })
                }
                data-testid="approved-checkbox"
              />
              <Label htmlFor="approved" className="font-normal">
                Approved (checked manufacturers appear in part selection)
              </Label>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)} disabled={saving}>
              Cancel
            </Button>
            <Button onClick={handleSave} disabled={saving} data-testid="save-btn">
              {saving ? "Saving..." : editingManufacturer ? "Update" : "Add"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent data-testid="delete-dialog">
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Manufacturer</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete <strong>{manufacturerToDelete?.name}</strong>?
              {" "}This action cannot be undone.
              {" "}<br /><br />
              <strong>Note:</strong> If this manufacturer is referenced by any parts, deletion will fail.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              disabled={deleting}
              className="bg-destructive hover:bg-destructive/90"
              data-testid="confirm-delete-btn"
            >
              {deleting ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
export default Vendors;
