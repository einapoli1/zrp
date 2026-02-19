import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";

const mockPO = {
  id: "PO-001",
  vendor_id: "V-001",
  status: "sent",
  created_at: "2025-01-20T00:00:00Z",
  expected_date: "2025-02-01T00:00:00Z",
  notes: "Rush order",
  lines: [
    { id: "L1", ipn: "IPN-001", mpn: "RC0805", manufacturer: "Yageo", qty_ordered: 1000, qty_received: 0, unit_price: 0.01 },
    { id: "L2", ipn: "IPN-002", mpn: "GRM21", manufacturer: "Murata", qty_ordered: 500, qty_received: 200, unit_price: 0.10 },
  ],
};

const mockVendor = {
  id: "V-001",
  name: "Acme Corp",
  contact_email: "john@acme.com",
  contact_phone: "555-1234",
  status: "active",
  created_at: "2024-01-01",
};

const mockGetPurchaseOrder = vi.fn().mockResolvedValue(mockPO);
const mockGetVendor = vi.fn().mockResolvedValue(mockVendor);

vi.mock("../lib/api", () => ({
  api: {
    getPurchaseOrder: (...args: any[]) => mockGetPurchaseOrder(...args),
    getVendor: (...args: any[]) => mockGetVendor(...args),
  },
}));

vi.mock("qrcode.react", () => ({
  QRCodeSVG: ({ value }: { value: string }) => <svg data-testid="qr-code" data-value={value} />,
}));

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useParams: () => ({ id: "PO-001" }) };
});

import POPrint from "./POPrint";

beforeEach(() => vi.clearAllMocks());

describe("POPrint", () => {
  it("renders PO details", async () => {
    render(<POPrint />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });
    expect(screen.getByText("sent")).toBeInTheDocument();
  });

  it("renders vendor info", async () => {
    render(<POPrint />);
    await waitFor(() => {
      expect(screen.getByText("Acme Corp")).toBeInTheDocument();
    });
    expect(screen.getByText("john@acme.com")).toBeInTheDocument();
    expect(screen.getByText("555-1234")).toBeInTheDocument();
  });

  it("renders notes", async () => {
    render(<POPrint />);
    await waitFor(() => {
      expect(screen.getByText("Rush order")).toBeInTheDocument();
    });
  });

  it("renders line items table", async () => {
    render(<POPrint />);
    await waitFor(() => {
      expect(screen.getByText("Line Items")).toBeInTheDocument();
    });
    expect(screen.getByText("IPN-001")).toBeInTheDocument();
    expect(screen.getByText("RC0805")).toBeInTheDocument();
    expect(screen.getByText("Yageo")).toBeInTheDocument();
    expect(screen.getByText("IPN-002")).toBeInTheDocument();
  });

  it("calculates total amount", async () => {
    render(<POPrint />);
    // Total: 1000*0.01 + 500*0.10 = 10 + 50 = 60
    await waitFor(() => {
      expect(screen.getByText("$60.00")).toBeInTheDocument();
    });
  });

  it("renders QR code", async () => {
    render(<POPrint />);
    await waitFor(() => {
      expect(screen.getByTestId("qr-code")).toBeInTheDocument();
    });
    expect(screen.getByTestId("qr-code").getAttribute("data-value")).toContain("/purchase-orders/PO-001");
  });

  it("has Print PO button", async () => {
    render(<POPrint />);
    await waitFor(() => {
      expect(screen.getByText("Print PO")).toBeInTheDocument();
    });
  });

  it("shows not found when PO missing", async () => {
    mockGetPurchaseOrder.mockRejectedValueOnce(new Error("Not found"));
    render(<POPrint />);
    await waitFor(() => {
      expect(screen.getByText("Purchase Order not found.")).toBeInTheDocument();
    });
  });

  it("renders signature area", async () => {
    render(<POPrint />);
    await waitFor(() => {
      expect(screen.getByText("Authorized By:")).toBeInTheDocument();
    });
    expect(screen.getByText("Received By:")).toBeInTheDocument();
  });
});
