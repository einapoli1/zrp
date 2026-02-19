import { useEffect, useState, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Separator } from "../components/ui/separator";
import { Skeleton } from "../components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import {
  ArrowLeft,
  FileText,
  Calendar,
  User,
  GitBranch,
  History,

  RotateCcw,
  CheckCircle,
} from "lucide-react";
import { api, type Document, type DocumentVersion, type DiffLine } from "../lib/api";

const statusConfig: Record<string, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
  draft: { label: "Draft", variant: "secondary" },
  released: { label: "Released", variant: "default" },
  approved: { label: "Approved", variant: "default" },
  obsolete: { label: "Obsolete", variant: "destructive" },
};

export default function DocumentDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [doc, setDoc] = useState<(Document & { attachments?: any[] }) | null>(null);
  const [versions, setVersions] = useState<DocumentVersion[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedVersion, setSelectedVersion] = useState<DocumentVersion | null>(null);
  const [diffFrom, setDiffFrom] = useState<string>("");
  const [diffTo, setDiffTo] = useState<string>("");
  const [diffLines, setDiffLines] = useState<DiffLine[]>([]);
  const [diffLoading, setDiffLoading] = useState(false);
  const [gitConfigured, setGitConfigured] = useState(false);

  const fetchDoc = useCallback(async () => {
    if (!id) return;
    try {
      const [docData, versionsData, gitCfg] = await Promise.all([
        api.getDocument(id),
        api.getDocumentVersions(id),
        api.getGitDocsSettings().catch(() => null),
      ]);
      setDoc(docData);
      setVersions(versionsData);
      setGitConfigured(!!gitCfg?.repo_url && gitCfg.repo_url !== "");
    } catch {
      // not found
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    fetchDoc();
  }, [fetchDoc]);

  const handleRelease = async () => {
    if (!id) return;
    await api.releaseDocument(id);
    fetchDoc();
  };

  const handleRevert = async (revision: string) => {
    if (!id) return;
    if (!confirm(`Revert document to revision ${revision}?`)) return;
    await api.revertDocument(id, revision);
    setSelectedVersion(null);
    fetchDoc();
  };

  const handleDiff = async () => {
    if (!id || !diffFrom || !diffTo) return;
    setDiffLoading(true);
    try {
      const result = await api.getDocumentDiff(id, diffFrom, diffTo);
      setDiffLines(result.lines);
    } catch {
      setDiffLines([]);
    } finally {
      setDiffLoading(false);
    }
  };

  const handlePushToGit = async () => {
    if (!id) return;
    try {
      const result = await api.pushDocumentToGit(id);
      alert(`Pushed to git: ${result.file}`);
    } catch (e: any) {
      alert(`Git push failed: ${e.message}`);
    }
  };

  if (loading) {
    return (
      <div className="space-y-4 p-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (!doc) {
    return (
      <div className="p-6">
        <p>Document not found</p>
        <Button variant="outline" onClick={() => navigate("/documents")}>
          <ArrowLeft className="mr-2 h-4 w-4" /> Back
        </Button>
      </div>
    );
  }

  const status = statusConfig[doc.status] || statusConfig.draft;
  const revisionOptions = [...new Set(versions.map((v) => v.revision))];

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Button variant="outline" size="sm" onClick={() => navigate("/documents")}>
            <ArrowLeft className="mr-2 h-4 w-4" /> Back
          </Button>
          <div>
            <div className="flex items-center space-x-2">
              <h1 className="text-2xl font-bold">{doc.title}</h1>
              <Badge variant={status.variant}>{status.label}</Badge>
              <Badge variant="outline">Rev {doc.revision}</Badge>
            </div>
            <p className="text-sm text-muted-foreground">{doc.id}</p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          {doc.status === "draft" && (
            <Button onClick={handleRelease} size="sm">
              <CheckCircle className="mr-2 h-4 w-4" /> Release
            </Button>
          )}
          {gitConfigured && (
            <Button variant="outline" size="sm" onClick={handlePushToGit}>
              <GitBranch className="mr-2 h-4 w-4" /> Push to Git
            </Button>
          )}
        </div>
      </div>

      <Tabs defaultValue="content">
        <TabsList>
          <TabsTrigger value="content">Content</TabsTrigger>
          <TabsTrigger value="versions">
            Version History ({versions.length})
          </TabsTrigger>
          <TabsTrigger value="diff">Diff Viewer</TabsTrigger>
        </TabsList>

        {/* Content Tab */}
        <TabsContent value="content">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center space-x-2">
                <FileText className="h-5 w-5" />
                <span>Document Content</span>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-3 gap-4 mb-4">
                <div className="flex items-center space-x-2">
                  <Calendar className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm">Category: {doc.category || "—"}</span>
                </div>
                <div className="flex items-center space-x-2">
                  <User className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm">Created by: {doc.created_by}</span>
                </div>
                <div className="flex items-center space-x-2">
                  <Calendar className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm">Updated: {doc.updated_at}</span>
                </div>
              </div>
              <Separator className="my-4" />
              <pre className="whitespace-pre-wrap font-mono text-sm bg-muted p-4 rounded-lg">
                {doc.content || "No content"}
              </pre>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Version History Tab */}
        <TabsContent value="versions">
          <div className="grid grid-cols-3 gap-4">
            <div className="col-span-1 space-y-2">
              <h3 className="font-semibold text-sm text-muted-foreground uppercase">
                Timeline
              </h3>
              {versions.length === 0 ? (
                <p className="text-sm text-muted-foreground">No versions yet</p>
              ) : (
                versions.map((v) => (
                  <Card
                    key={v.id}
                    className={`cursor-pointer transition-colors ${
                      selectedVersion?.id === v.id ? "border-primary" : ""
                    }`}
                    onClick={() => setSelectedVersion(v)}
                  >
                    <CardContent className="p-3">
                      <div className="flex items-center justify-between">
                        <Badge variant="outline">Rev {v.revision}</Badge>
                        <Badge
                          variant={
                            v.status === "released"
                              ? "default"
                              : v.status === "obsolete"
                              ? "destructive"
                              : "secondary"
                          }
                        >
                          {v.status}
                        </Badge>
                      </div>
                      <p className="text-xs text-muted-foreground mt-1">
                        {v.created_at} by {v.created_by}
                      </p>
                      {v.change_summary && (
                        <p className="text-xs mt-1">{v.change_summary}</p>
                      )}
                      {v.eco_id && (
                        <Badge variant="outline" className="mt-1 text-xs">
                          ECO: {v.eco_id}
                        </Badge>
                      )}
                    </CardContent>
                  </Card>
                ))
              )}
            </div>

            <div className="col-span-2">
              {selectedVersion ? (
                <Card>
                  <CardHeader>
                    <div className="flex items-center justify-between">
                      <CardTitle>
                        Revision {selectedVersion.revision}
                      </CardTitle>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleRevert(selectedVersion.revision)}
                      >
                        <RotateCcw className="mr-2 h-4 w-4" /> Revert to this
                        version
                      </Button>
                    </div>
                  </CardHeader>
                  <CardContent>
                    <pre className="whitespace-pre-wrap font-mono text-sm bg-muted p-4 rounded-lg max-h-96 overflow-y-auto">
                      {selectedVersion.content || "No content"}
                    </pre>
                  </CardContent>
                </Card>
              ) : (
                <div className="flex items-center justify-center h-64 text-muted-foreground">
                  <div className="text-center">
                    <History className="h-12 w-12 mx-auto mb-2 opacity-50" />
                    <p>Select a version to view its content</p>
                  </div>
                </div>
              )}
            </div>
          </div>
        </TabsContent>

        {/* Diff Viewer Tab */}
        <TabsContent value="diff">
          <Card>
            <CardHeader>
              <CardTitle>Compare Revisions</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center space-x-4 mb-4">
                <select
                  className="border rounded px-2 py-1 text-sm"
                  value={diffFrom}
                  onChange={(e) => setDiffFrom(e.target.value)}
                >
                  <option value="">From revision...</option>
                  {revisionOptions.map((r) => (
                    <option key={r} value={r}>
                      Rev {r}
                    </option>
                  ))}
                </select>
                <span className="text-muted-foreground">→</span>
                <select
                  className="border rounded px-2 py-1 text-sm"
                  value={diffTo}
                  onChange={(e) => setDiffTo(e.target.value)}
                >
                  <option value="">To revision...</option>
                  {revisionOptions.map((r) => (
                    <option key={r} value={r}>
                      Rev {r}
                    </option>
                  ))}
                </select>
                <Button
                  size="sm"
                  onClick={handleDiff}
                  disabled={!diffFrom || !diffTo || diffLoading}
                >
                  Compare
                </Button>
              </div>

              {diffLoading && <Skeleton className="h-40 w-full" />}

              {diffLines.length > 0 && (
                <div className="font-mono text-sm border rounded-lg overflow-hidden">
                  {diffLines.map((line, i) => (
                    <div
                      key={i}
                      className={`px-4 py-0.5 ${
                        line.type === "added"
                          ? "bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300"
                          : line.type === "removed"
                          ? "bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300"
                          : ""
                      }`}
                    >
                      <span className="select-none mr-2 text-muted-foreground">
                        {line.type === "added"
                          ? "+"
                          : line.type === "removed"
                          ? "-"
                          : " "}
                      </span>
                      {line.text}
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
