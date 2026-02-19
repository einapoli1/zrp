import { useEffect, useState } from "react";
import { api, type EmailLogEntry } from "../lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Mail, RefreshCw } from "lucide-react";
import { LoadingState } from "../components/LoadingState";
import { EmptyState } from "../components/EmptyState";
import { ErrorState } from "../components/ErrorState";

export default function EmailLog() {
  const [entries, setEntries] = useState<EmailLogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchEmailLog = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.getEmailLog();
      setEntries(Array.isArray(data) ? data : []);
    } catch (err: any) {
      setError(err.message || "Failed to load email log");
      setEntries([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchEmailLog();
  }, []);

  if (loading) {
    return <LoadingState variant="spinner" message="Loading email log..." />;
  }

  if (error) {
    return (
      <div className="p-6">
        <ErrorState
          title="Failed to load email log"
          message={error}
          onRetry={fetchEmailLog}
        />
      </div>
    );
  }

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
            <Mail className="h-8 w-8" /> Email Log
          </h1>
          <p className="text-muted-foreground">
            Admin view of all sent email notifications.
          </p>
        </div>
        <Button variant="outline" onClick={fetchEmailLog}>
          <RefreshCw className="h-4 w-4 mr-2" />
          Refresh
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Sent Emails ({entries.length})</CardTitle>
        </CardHeader>
        <CardContent>
          {entries.length === 0 ? (
            <EmptyState
              icon={Mail}
              title="No emails sent yet"
              description="Email notifications will appear here when they are sent from the system."
            />
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b">
                    <th className="text-left p-2">To</th>
                    <th className="text-left p-2 hidden md:table-cell">Subject</th>
                    <th className="text-left p-2 hidden lg:table-cell">Event</th>
                    <th className="text-left p-2">Status</th>
                    <th className="text-left p-2 hidden xl:table-cell">Error</th>
                    <th className="text-left p-2 hidden sm:table-cell">Sent At</th>
                  </tr>
                </thead>
                <tbody>
                  {entries.map((e) => (
                    <tr key={e.id} className="border-b hover:bg-muted/50">
                      <td className="p-2 font-medium">{e.to_address}</td>
                      <td className="p-2 hidden md:table-cell max-w-xs truncate">
                        {e.subject}
                      </td>
                      <td className="p-2 hidden lg:table-cell">
                        {e.event_type ? (
                          <Badge variant="outline">{e.event_type}</Badge>
                        ) : (
                          <span className="text-muted-foreground">—</span>
                        )}
                      </td>
                      <td className="p-2">
                        <Badge variant={e.status === "sent" ? "default" : "destructive"}>
                          {e.status}
                        </Badge>
                      </td>
                      <td className="p-2 text-destructive hidden xl:table-cell">
                        {e.error || "—"}
                      </td>
                      <td className="p-2 text-muted-foreground hidden sm:table-cell">
                        {e.sent_at}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
