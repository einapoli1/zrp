import { useEffect, useState } from "react";
import { api, EmailLogEntry } from "../lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Mail } from "lucide-react";

export default function EmailLog() {
  const [entries, setEntries] = useState<EmailLogEntry[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getEmailLog().then((data) => {
      setEntries(Array.isArray(data) ? data : []);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, []);

  if (loading) return <div className="p-6">Loading...</div>;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-1 flex items-center gap-2">
        <Mail className="h-6 w-6" /> Email Log
      </h1>
      <p className="text-muted-foreground mb-6">Admin view of all sent email notifications.</p>

      <Card>
        <CardHeader>
          <CardTitle>Sent Emails ({entries.length})</CardTitle>
        </CardHeader>
        <CardContent>
          {entries.length === 0 ? (
            <p className="text-muted-foreground">No emails sent yet.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b">
                    <th className="text-left p-2">To</th>
                    <th className="text-left p-2">Subject</th>
                    <th className="text-left p-2">Event</th>
                    <th className="text-left p-2">Status</th>
                    <th className="text-left p-2">Error</th>
                    <th className="text-left p-2">Sent At</th>
                  </tr>
                </thead>
                <tbody>
                  {entries.map((e) => (
                    <tr key={e.id} className="border-b">
                      <td className="p-2">{e.to_address}</td>
                      <td className="p-2">{e.subject}</td>
                      <td className="p-2">
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
                      <td className="p-2 text-red-500">{e.error || "—"}</td>
                      <td className="p-2">{e.sent_at}</td>
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
