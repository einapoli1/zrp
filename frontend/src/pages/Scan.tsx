import { useState, useCallback, lazy, Suspense } from "react";
import { useNavigate } from "react-router-dom";
import { ScanLine, Loader2, Search } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { EmptyState } from "../components/EmptyState";
import { ErrorState } from "../components/ErrorState";

// Lazy load BarcodeScanner to reduce initial bundle size (329KB chunk)
const BarcodeScanner = lazy(() => import("../components/BarcodeScanner").then(m => ({ default: m.BarcodeScanner })));

interface ScanResult {
  type: string;
  id: string;
  label: string;
  link: string;
}

function Scan() {
  const navigate = useNavigate();
  const [results, setResults] = useState<ScanResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [scannedCode, setScannedCode] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleScan = useCallback(async (code: string) => {
    setScannedCode(code);
    setLoading(true);
    setError(null);
    try {
      const resp = await fetch(`/api/v1/scan/${encodeURIComponent(code)}`);
      if (!resp.ok) throw new Error("Lookup failed");
      const data = await resp.json();
      const matches: ScanResult[] = data.results || [];
      setResults(matches);
      // Auto-navigate if single result
      if (matches.length === 1) {
        navigate(matches[0].link);
      }
    } catch (err: any) {
      setError(err.message || "Could not look up scanned code");
      setResults([]);
    } finally {
      setLoading(false);
    }
  }, [navigate]);

  const handleRetry = () => {
    setError(null);
    setResults([]);
    setScannedCode(null);
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
          <ScanLine className="h-8 w-8" />
          Barcode Scanner
        </h1>
        <p className="text-muted-foreground">
          Scan a barcode or QR code to find parts, inventory, or devices.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Scan Barcode</CardTitle>
        </CardHeader>
        <CardContent>
          <Suspense fallback={
            <div className="flex items-center justify-center p-8" role="status" aria-label="Loading scanner">
              <Loader2 className="h-8 w-8 animate-spin" />
            </div>
          }>
            <BarcodeScanner onScan={handleScan} />
          </Suspense>
        </CardContent>
      </Card>

      {loading && (
        <div className="flex items-center gap-2 text-muted-foreground" role="status">
          <Loader2 className="h-4 w-4 animate-spin" />
          <span>Looking up "{scannedCode}"...</span>
        </div>
      )}

      {error && (
        <ErrorState
          variant="inline"
          title="Lookup failed"
          message={error}
          onRetry={handleRetry}
        />
      )}

      {!loading && results.length > 1 && (
        <Card>
          <CardHeader>
            <CardTitle>Results for "{scannedCode}"</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {results.map((r, i) => (
              <Button
                key={i}
                variant="outline"
                className="w-full justify-start gap-2"
                onClick={() => navigate(r.link)}
              >
                <Badge variant="secondary">{r.type}</Badge>
                <span className="truncate">{r.label}</span>
              </Button>
            ))}
          </CardContent>
        </Card>
      )}

      {!loading && scannedCode && results.length === 0 && !error && (
        <EmptyState
          icon={Search}
          title="No matches found"
          description={`No parts, inventory, or devices found matching "${scannedCode}". Try scanning a different code.`}
        />
      )}
    </div>
  );
}

export default Scan;
