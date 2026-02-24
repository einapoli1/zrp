import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "../components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../components/ui/table";
import { Plus, Trash2, Settings, Play, Eye, Save, X } from "lucide-react";
import { toast } from "sonner";
import { LoadingState } from "../components/LoadingState";
import { EmptyState } from "../components/EmptyState";
import { ErrorState } from "../components/ErrorState";

interface ConfigurationTemplate {
  id: number;
  name: string;
  model_format: string;
  created_at: string;
  updated_at: string;
  parameters?: ConfigurationParameter[];
  parts?: ConfigurationPart[];
}

interface ConfigurationParameter {
  id: number;
  template_id: number;
  name: string;
  type: "enum" | "range";
  values_json: string;
  created_at: string;
}

interface ConfigurationPart {
  id: number;
  template_id: number;
  ipn: string;
  quantity: number;
  include_all_variants: number;
  constraints_json: string;
  created_at: string;
  description?: string;
}

function Configurator() {
  const navigate = useNavigate();
  const [templates, setTemplates] = useState<ConfigurationTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState("list");

  // Template editor state
  const [editingTemplate, setEditingTemplate] = useState<ConfigurationTemplate | null>(null);
  const [templateName, setTemplateName] = useState("");
  const [modelFormat, setModelFormat] = useState("");

  // Parameter state
  const [paramName, setParamName] = useState("");
  const [paramType, setParamType] = useState<"enum" | "range">("enum");
  const [enumValues, setEnumValues] = useState<string[]>([]);
  const [enumInput, setEnumInput] = useState("");
  const [rangeMin, setRangeMin] = useState("");
  const [rangeMax, setRangeMax] = useState("");
  const [rangeUnit, setRangeUnit] = useState("");

  // Part state
  const [partSearchOpen, setPartSearchOpen] = useState(false);
  const [partSearch, setPartSearch] = useState("");
  const [partQuantity, setPartQuantity] = useState(1);
  const [partIncludeAll, setPartIncludeAll] = useState(false);
  const [editingConstraints, setEditingConstraints] = useState<ConfigurationPart | null>(null);
  const [constraintValues, setConstraintValues] = useState<Record<string, any>>({});

  // Preview/Generate state
  const [selectedTemplateId, setSelectedTemplateId] = useState<number | null>(null);
  const [previewData, setPreviewData] = useState<any[]>([]);
  const [generating, setGenerating] = useState(false);

  useEffect(() => {
    fetchTemplates();
  }, []);

  const fetchTemplates = async () => {
    try {
      setLoading(true);
      const response = await fetch("/api/v1/configurator/templates");
      if (!response.ok) throw new Error("Failed to fetch templates");
      const data = await response.json();
      setTemplates(data);
      setError(null);
    } catch (err) {
      setError((err as Error).message);
      toast.error("Failed to load templates");
    } finally {
      setLoading(false);
    }
  };

  const createNewTemplate = () => {
    setEditingTemplate({
      id: 0,
      name: "",
      model_format: "",
      created_at: "",
      updated_at: "",
      parameters: [],
      parts: [],
    });
    setTemplateName("");
    setModelFormat("");
    setActiveTab("editor");
  };

  const editTemplate = async (id: number) => {
    try {
      const response = await fetch(`/api/v1/configurator/templates/${id}`);
      if (!response.ok) throw new Error("Failed to fetch template");
      const data = await response.json();
      setEditingTemplate(data);
      setTemplateName(data.name);
      setModelFormat(data.model_format);
      setActiveTab("editor");
    } catch (err) {
      toast.error("Failed to load template");
    }
  };

  const saveTemplate = async () => {
    if (!templateName || !modelFormat) {
      toast.error("Name and model format are required");
      return;
    }

    if (!modelFormat.includes("{") || !modelFormat.includes("}")) {
      toast.error("Model format must contain at least one {param} placeholder");
      return;
    }

    try {
      const payload = { name: templateName, model_format: modelFormat };
      
      if (editingTemplate && editingTemplate.id > 0) {
        // Update existing
        const response = await fetch(`/api/v1/configurator/templates/${editingTemplate.id}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        });
        if (!response.ok) throw new Error("Failed to update template");
        const updated = await response.json();
        setEditingTemplate(updated);
        toast.success("Template updated");
      } else {
        // Create new
        const response = await fetch("/api/v1/configurator/templates", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        });
        if (!response.ok) throw new Error("Failed to create template");
        const created = await response.json();
        setEditingTemplate(created);
        toast.success("Template created");
      }
      
      fetchTemplates();
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const deleteTemplate = async (id: number) => {
    if (!confirm("Delete this template?")) return;

    try {
      const response = await fetch(`/api/v1/configurator/templates/${id}`, {
        method: "DELETE",
      });
      if (!response.ok) throw new Error("Failed to delete template");
      toast.success("Template deleted");
      fetchTemplates();
      if (editingTemplate?.id === id) {
        setEditingTemplate(null);
        setActiveTab("list");
      }
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  // Parameters

  const addParameter = async () => {
    if (!editingTemplate || editingTemplate.id === 0) {
      toast.error("Save template first");
      return;
    }

    if (!paramName) {
      toast.error("Parameter name is required");
      return;
    }

    let valuesJson = "";
    if (paramType === "enum") {
      if (enumValues.length === 0) {
        toast.error("Add at least one enum value");
        return;
      }
      valuesJson = JSON.stringify(enumValues);
    } else {
      if (!rangeMin || !rangeMax) {
        toast.error("Range min and max are required");
        return;
      }
      valuesJson = JSON.stringify({
        min: parseFloat(rangeMin),
        max: parseFloat(rangeMax),
        unit: rangeUnit,
      });
    }

    try {
      const response = await fetch(`/api/v1/configurator/templates/${editingTemplate.id}/parameters`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: paramName,
          type: paramType,
          values_json: valuesJson,
        }),
      });
      if (!response.ok) throw new Error("Failed to add parameter");
      
      // Refresh template
      const refreshed = await fetch(`/api/v1/configurator/templates/${editingTemplate.id}`);
      const data = await refreshed.json();
      setEditingTemplate(data);
      
      // Reset form
      setParamName("");
      setEnumValues([]);
      setRangeMin("");
      setRangeMax("");
      setRangeUnit("");
      
      toast.success("Parameter added");
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const deleteParameter = async (paramId: number) => {
    try {
      const response = await fetch(`/api/v1/configurator/parameters/${paramId}`, {
        method: "DELETE",
      });
      if (!response.ok) throw new Error("Failed to delete parameter");
      
      // Refresh template
      if (editingTemplate) {
        const refreshed = await fetch(`/api/v1/configurator/templates/${editingTemplate.id}`);
        const data = await refreshed.json();
        setEditingTemplate(data);
      }
      
      toast.success("Parameter deleted");
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  // Parts

  const addPart = async (ipn: string) => {
    if (!editingTemplate || editingTemplate.id === 0) {
      toast.error("Save template first");
      return;
    }

    try {
      const response = await fetch(`/api/v1/configurator/templates/${editingTemplate.id}/parts`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          ipn,
          quantity: partQuantity,
          include_all_variants: partIncludeAll ? 1 : 0,
          constraints_json: "{}",
        }),
      });
      if (!response.ok) throw new Error("Failed to add part");
      
      // Refresh template
      const refreshed = await fetch(`/api/v1/configurator/templates/${editingTemplate.id}`);
      const data = await refreshed.json();
      setEditingTemplate(data);
      
      setPartSearchOpen(false);
      setPartSearch("");
      setPartQuantity(1);
      setPartIncludeAll(false);
      
      toast.success("Part added");
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const deletePart = async (partId: number) => {
    try {
      const response = await fetch(`/api/v1/configurator/parts/${partId}`, {
        method: "DELETE",
      });
      if (!response.ok) throw new Error("Failed to delete part");
      
      // Refresh template
      if (editingTemplate) {
        const refreshed = await fetch(`/api/v1/configurator/templates/${editingTemplate.id}`);
        const data = await refreshed.json();
        setEditingTemplate(data);
      }
      
      toast.success("Part removed");
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const saveConstraints = async () => {
    if (!editingConstraints) return;

    try {
      const response = await fetch(`/api/v1/configurator/parts/${editingConstraints.id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          quantity: editingConstraints.quantity,
          include_all_variants: editingConstraints.include_all_variants,
          constraints_json: JSON.stringify(constraintValues),
        }),
      });
      if (!response.ok) throw new Error("Failed to update constraints");
      
      // Refresh template
      if (editingTemplate) {
        const refreshed = await fetch(`/api/v1/configurator/templates/${editingTemplate.id}`);
        const data = await refreshed.json();
        setEditingTemplate(data);
      }
      
      setEditingConstraints(null);
      toast.success("Constraints updated");
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  // Preview & Generate

  const previewVariants = async () => {
    if (!selectedTemplateId) {
      toast.error("Select a template");
      return;
    }

    try {
      const response = await fetch(`/api/v1/configurator/templates/${selectedTemplateId}/preview`);
      if (!response.ok) throw new Error("Failed to preview");
      const data = await response.json();
      setPreviewData(data.preview || []);
      toast.success(`Preview: ${data.total_count} variants`);
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const generateVariants = async () => {
    if (!selectedTemplateId) {
      toast.error("Select a template");
      return;
    }

    if (!confirm("This will create an ECO with all variants. Continue?")) return;

    try {
      setGenerating(true);
      const response = await fetch(`/api/v1/configurator/templates/${selectedTemplateId}/generate`, {
        method: "POST",
      });
      if (!response.ok) throw new Error("Failed to generate");
      const data = await response.json();
      toast.success(`Generated ${data.variant_count} variants in ${data.eco_id}`);
      navigate(`/ecos/${data.eco_id}`);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setGenerating(false);
    }
  };

  if (loading) return <LoadingState />;
  if (error) return <ErrorState message={error} onRetry={fetchTemplates} />;

  return (
    <div className="container mx-auto p-6 max-w-7xl">
      <div className="mb-8">
        <h1 className="text-3xl font-bold mb-2">Product Configurator</h1>
        <p className="text-gray-600">
          Define product variants and generate BOMs via ECO process
        </p>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="list">Templates</TabsTrigger>
          <TabsTrigger value="editor">Editor</TabsTrigger>
          <TabsTrigger value="generate">Preview & Generate</TabsTrigger>
        </TabsList>

        {/* Tab 1: Templates List */}
        <TabsContent value="list" className="space-y-4">
          <div className="flex justify-between items-center">
            <h2 className="text-xl font-semibold">Configuration Templates</h2>
            <Button onClick={createNewTemplate}>
              <Plus className="h-4 w-4 mr-2" />
              New Template
            </Button>
          </div>

          {templates.length === 0 ? (
            <EmptyState
              icon={Settings}
              message="No templates yet"
              action={{ label: "Create Template", onClick: createNewTemplate }}
            />
          ) : (
            <Card>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Model Format</TableHead>
                    <TableHead className="text-center">Parameters</TableHead>
                    <TableHead className="text-center">Parts</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {templates.map((t) => (
                    <TableRow key={t.id}>
                      <TableCell className="font-medium">{t.name}</TableCell>
                      <TableCell className="font-mono text-sm">{t.model_format}</TableCell>
                      <TableCell className="text-center">
                        <Badge variant="outline">{t.parameters?.length || 0}</Badge>
                      </TableCell>
                      <TableCell className="text-center">
                        <Badge variant="outline">{t.parts?.length || 0}</Badge>
                      </TableCell>
                      <TableCell className="text-right space-x-2">
                        <Button size="sm" variant="outline" onClick={() => editTemplate(t.id)}>
                          Edit
                        </Button>
                        <Button size="sm" variant="destructive" onClick={() => deleteTemplate(t.id)}>
                          Delete
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </Card>
          )}
        </TabsContent>

        {/* Tab 2: Template Editor */}
        <TabsContent value="editor" className="space-y-6">
          {!editingTemplate ? (
            <EmptyState
              icon={Settings}
              message="Select or create a template to edit"
              action={{ label: "New Template", onClick: createNewTemplate }}
            />
          ) : (
            <>
              {/* Template Info */}
              <Card>
                <CardHeader>
                  <CardTitle>Template Details</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium mb-1">Name</label>
                    <Input
                      value={templateName}
                      onChange={(e) => setTemplateName(e.target.value)}
                      placeholder="e.g., uATS 1.2kVA"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1">Model Format</label>
                    <Input
                      value={modelFormat}
                      onChange={(e) => setModelFormat(e.target.value)}
                      placeholder="e.g., PCA-uATS-{voltage}-{amperage}-{length}"
                      className="font-mono"
                    />
                    <p className="text-xs text-gray-500 mt-1">
                      Use &#123;parameter_name&#125; for placeholders
                    </p>
                  </div>
                  <div className="flex gap-2">
                    <Button onClick={saveTemplate}>
                      <Save className="h-4 w-4 mr-2" />
                      Save Template
                    </Button>
                    <Button variant="outline" onClick={() => {
                      setEditingTemplate(null);
                      setActiveTab("list");
                    }}>
                      <X className="h-4 w-4 mr-2" />
                      Cancel
                    </Button>
                  </div>
                </CardContent>
              </Card>

              {/* Parameters */}
              {editingTemplate.id > 0 && (
                <Card>
                  <CardHeader>
                    <CardTitle>Parameters</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    {editingTemplate.parameters && editingTemplate.parameters.length > 0 && (
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>Name</TableHead>
                            <TableHead>Type</TableHead>
                            <TableHead>Values</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {editingTemplate.parameters.map((p) => (
                            <TableRow key={p.id}>
                              <TableCell className="font-medium">{p.name}</TableCell>
                              <TableCell>
                                <Badge variant={p.type === "enum" ? "default" : "secondary"}>
                                  {p.type}
                                </Badge>
                              </TableCell>
                              <TableCell className="font-mono text-sm">
                                {p.values_json}
                              </TableCell>
                              <TableCell className="text-right">
                                <Button
                                  size="sm"
                                  variant="destructive"
                                  onClick={() => deleteParameter(p.id)}
                                >
                                  <Trash2 className="h-3 w-3" />
                                </Button>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    )}

                    <div className="border-t pt-4 space-y-3">
                      <h3 className="font-medium">Add Parameter</h3>
                      <div className="grid grid-cols-2 gap-4">
                        <div>
                          <label className="block text-sm mb-1">Name</label>
                          <Input
                            value={paramName}
                            onChange={(e) => setParamName(e.target.value)}
                            placeholder="voltage"
                          />
                        </div>
                        <div>
                          <label className="block text-sm mb-1">Type</label>
                          <Select value={paramType} onValueChange={(v) => setParamType(v as "enum" | "range")}>
                            <SelectTrigger>
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="enum">Enum</SelectItem>
                              <SelectItem value="range">Range</SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                      </div>

                      {paramType === "enum" ? (
                        <div>
                          <label className="block text-sm mb-1">Values</label>
                          <div className="flex gap-2 mb-2">
                            <Input
                              value={enumInput}
                              onChange={(e) => setEnumInput(e.target.value)}
                              placeholder="120V"
                              onKeyPress={(e) => {
                                if (e.key === "Enter") {
                                  if (enumInput) {
                                    setEnumValues([...enumValues, enumInput]);
                                    setEnumInput("");
                                  }
                                }
                              }}
                            />
                            <Button
                              onClick={() => {
                                if (enumInput) {
                                  setEnumValues([...enumValues, enumInput]);
                                  setEnumInput("");
                                }
                              }}
                            >
                              Add
                            </Button>
                          </div>
                          <div className="flex flex-wrap gap-2">
                            {enumValues.map((v, i) => (
                              <Badge key={i} variant="secondary">
                                {v}
                                <button
                                  className="ml-2 text-xs"
                                  onClick={() => setEnumValues(enumValues.filter((_, idx) => idx !== i))}
                                >
                                  ×
                                </button>
                              </Badge>
                            ))}
                          </div>
                        </div>
                      ) : (
                        <div className="grid grid-cols-3 gap-4">
                          <div>
                            <label className="block text-sm mb-1">Min</label>
                            <Input
                              type="number"
                              value={rangeMin}
                              onChange={(e) => setRangeMin(e.target.value)}
                            />
                          </div>
                          <div>
                            <label className="block text-sm mb-1">Max</label>
                            <Input
                              type="number"
                              value={rangeMax}
                              onChange={(e) => setRangeMax(e.target.value)}
                            />
                          </div>
                          <div>
                            <label className="block text-sm mb-1">Unit</label>
                            <Input
                              value={rangeUnit}
                              onChange={(e) => setRangeUnit(e.target.value)}
                              placeholder="V"
                            />
                          </div>
                        </div>
                      )}

                      <Button onClick={addParameter}>
                        <Plus className="h-4 w-4 mr-2" />
                        Add Parameter
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              )}

              {/* Parts Pool */}
              {editingTemplate.id > 0 && (
                <Card>
                  <CardHeader>
                    <CardTitle>Parts Pool</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    {editingTemplate.parts && editingTemplate.parts.length > 0 && (
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>IPN</TableHead>
                            <TableHead>Description</TableHead>
                            <TableHead className="text-center">Quantity</TableHead>
                            <TableHead className="text-center">All Variants</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {editingTemplate.parts.map((p) => (
                            <TableRow key={p.id}>
                              <TableCell className="font-mono">{p.ipn}</TableCell>
                              <TableCell>{p.description}</TableCell>
                              <TableCell className="text-center">{p.quantity}</TableCell>
                              <TableCell className="text-center">
                                {p.include_all_variants ? "✓" : ""}
                              </TableCell>
                              <TableCell className="text-right space-x-2">
                                <Button
                                  size="sm"
                                  variant="outline"
                                  onClick={() => {
                                    setEditingConstraints(p);
                                    setConstraintValues(JSON.parse(p.constraints_json || "{}"));
                                  }}
                                >
                                  Constraints
                                </Button>
                                <Button
                                  size="sm"
                                  variant="destructive"
                                  onClick={() => deletePart(p.id)}
                                >
                                  <Trash2 className="h-3 w-3" />
                                </Button>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    )}

                    <Button onClick={() => setPartSearchOpen(true)}>
                      <Plus className="h-4 w-4 mr-2" />
                      Add Part
                    </Button>
                  </CardContent>
                </Card>
              )}
            </>
          )}
        </TabsContent>

        {/* Tab 3: Preview & Generate */}
        <TabsContent value="generate" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Generate Variants</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">Select Template</label>
                <Select
                  value={selectedTemplateId?.toString()}
                  onValueChange={(v) => setSelectedTemplateId(parseInt(v))}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Choose template..." />
                  </SelectTrigger>
                  <SelectContent>
                    {templates.map((t) => (
                      <SelectItem key={t.id} value={t.id.toString()}>
                        {t.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="flex gap-2">
                <Button variant="outline" onClick={previewVariants}>
                  <Eye className="h-4 w-4 mr-2" />
                  Preview (First 10)
                </Button>
                <Button onClick={generateVariants} disabled={generating}>
                  <Play className="h-4 w-4 mr-2" />
                  {generating ? "Generating..." : "Generate All Variants"}
                </Button>
              </div>

              {previewData.length > 0 && (
                <div>
                  <h3 className="font-medium mb-2">Preview</h3>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Generated IPN</TableHead>
                        <TableHead className="text-right">BOM Items</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {previewData.map((v, i) => (
                        <TableRow key={i}>
                          <TableCell className="font-mono">{v.ipn}</TableCell>
                          <TableCell className="text-right">{v.bom_count}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Part Search Dialog */}
      <Dialog open={partSearchOpen} onOpenChange={setPartSearchOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add Part</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <Input
              placeholder="Search parts..."
              value={partSearch}
              onChange={(e) => setPartSearch(e.target.value)}
            />
            <div>
              <label className="block text-sm mb-1">Quantity</label>
              <Input
                type="number"
                min="1"
                value={partQuantity}
                onChange={(e) => setPartQuantity(parseInt(e.target.value))}
              />
            </div>
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={partIncludeAll}
                onChange={(e) => setPartIncludeAll(e.target.checked)}
              />
              <label className="text-sm">Include in all variants</label>
            </div>
            <Button
              onClick={() => addPart(partSearch)}
              disabled={!partSearch}
            >
              Add Part
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Constraints Dialog */}
      <Dialog open={!!editingConstraints} onOpenChange={() => setEditingConstraints(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Constraints - {editingConstraints?.ipn}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <p className="text-sm text-gray-600">
              Define parameter constraints for when this part should be included
            </p>
            {editingTemplate?.parameters?.map((param) => (
              <div key={param.id}>
                <label className="block text-sm font-medium mb-1">{param.name}</label>
                {param.type === "enum" ? (
                  <Select
                    value={constraintValues[param.name] || ""}
                    onValueChange={(v) =>
                      setConstraintValues({ ...constraintValues, [param.name]: v })
                    }
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Any" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="">Any</SelectItem>
                      {JSON.parse(param.values_json).map((v: string) => (
                        <SelectItem key={v} value={v}>
                          {v}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                ) : (
                  <div className="flex gap-2">
                    <Input placeholder="Min" type="number" />
                    <Input placeholder="Max" type="number" />
                  </div>
                )}
              </div>
            ))}
            <Button onClick={saveConstraints}>Save Constraints</Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

export default Configurator;
