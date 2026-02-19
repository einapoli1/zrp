import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import { mockWorkOrders } from "../test/mocks";

const mockWO = { ...mockWorkOrders[1], notes: "Test notes here" }; // WO-002, in_progress

const mockBOM = {
  wo_id: "WO-002",
  assembly_ipn: "IPN-003",
  qty: 5,
  bom: [
    { ipn: "IPN-001", description: "10k Resistor", qty_required: 50, qty_on_hand: 500, shortage: 0, status: "ok" },
    { ipn: "IPN-002", description: "100uF Cap", qty_required: 25, qty_on_hand: 10, shortage: 15, status: "shortage" },
  ],
};

const mockGetWorkOrder = vi.fn().mockResolvedValue(mockWO);
const mockGetWorkOrderBOM = vi.fn().mockResolvedValue(mockBOM);

vi.mock("../lib/api", () => ({
  api: {
    getWorkOrder: (...args: any[]) => mockGetWorkOrder(...args),
    getWorkOrderBOM: (...args: any[]) => mockGetWorkOrderBOM(...args),
  },
}));

vi.mock("qrcode.react", () => ({
  QRCodeSVG: ({ value }: { value: string }) => <svg data-testid="qr-code" data-value={value} />,
}));

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useParams: () => ({ id: "WO-002" }) };
});

import WorkOrderPrint from "./WorkOrderPrint";

beforeEach(() => vi.clearAllMocks());

describe("WorkOrderPrint", () => {
  it("renders work order details", async () => {
    render(<WorkOrderPrint />);
    await waitFor(() => {
      expect(screen.getByText("WO-002")).toBeInTheDocument();
    });
    expect(screen.getByText("IPN-003")).toBeInTheDocument();
    expect(screen.getByText("5")).toBeInTheDocument();
  });

  it("renders notes section", async () => {
    render(<WorkOrderPrint />);
    await waitFor(() => {
      expect(screen.getByText("Test notes here")).toBeInTheDocument();
    });
  });

  it("renders BOM table", async () => {
    render(<WorkOrderPrint />);
    await waitFor(() => {
      expect(screen.getByText("Bill of Materials")).toBeInTheDocument();
    });
    expect(screen.getByText("10k Resistor")).toBeInTheDocument();
    expect(screen.getByText("100uF Cap")).toBeInTheDocument();
  });

  it("renders QR code", async () => {
    render(<WorkOrderPrint />);
    await waitFor(() => {
      expect(screen.getByTestId("qr-code")).toBeInTheDocument();
    });
    expect(screen.getByTestId("qr-code").getAttribute("data-value")).toContain("/work-orders/WO-002");
  });

  it("renders sign-off area", async () => {
    render(<WorkOrderPrint />);
    await waitFor(() => {
      expect(screen.getByText("Sign-Off")).toBeInTheDocument();
    });
    expect(screen.getByText("Prepared By:")).toBeInTheDocument();
    expect(screen.getByText("QC Inspection:")).toBeInTheDocument();
  });

  it("has Print Traveler button", async () => {
    render(<WorkOrderPrint />);
    await waitFor(() => {
      expect(screen.getByText("Print Traveler")).toBeInTheDocument();
    });
  });

  it("shows not found when WO missing", async () => {
    mockGetWorkOrder.mockRejectedValueOnce(new Error("Not found"));
    // getWorkOrderBOM also rejects since .catch(() => null)
    mockGetWorkOrderBOM.mockRejectedValueOnce(new Error("Not found"));
    render(<WorkOrderPrint />);
    await waitFor(() => {
      expect(screen.getByText("Work Order not found.")).toBeInTheDocument();
    });
  });
});
