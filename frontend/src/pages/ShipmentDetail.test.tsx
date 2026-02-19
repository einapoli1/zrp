import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import { mockShipments } from "../test/mocks";

const mockGetShipment = vi.fn().mockResolvedValue(mockShipments[0]);
const mockShipShipment = vi.fn().mockResolvedValue({ ...mockShipments[0], status: "shipped", tracking_number: "1Z999", carrier: "UPS" });
const mockDeliverShipment = vi.fn().mockResolvedValue({ ...mockShipments[0], status: "delivered" });

vi.mock("../lib/api", () => ({
  api: {
    getShipment: (...args: any[]) => mockGetShipment(...args),
    shipShipment: (...args: any[]) => mockShipShipment(...args),
    deliverShipment: (...args: any[]) => mockDeliverShipment(...args),
  },
}));

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useNavigate: () => mockNavigate, useParams: () => ({ id: "SHP-2024-0001" }) };
});

import ShipmentDetail from "./ShipmentDetail";

beforeEach(() => vi.clearAllMocks());

describe("ShipmentDetail", () => {
  it("renders loading state", () => {
    render(<ShipmentDetail />);
    expect(screen.getByText("Loading shipment...")).toBeInTheDocument();
  });

  it("renders shipment details after loading", async () => {
    render(<ShipmentDetail />);
    await waitFor(() => {
      expect(screen.getByText("SHP-2024-0001")).toBeInTheDocument();
    });
    expect(screen.getByText(/123 Main St/)).toBeInTheDocument();
    expect(screen.getByText(/456 Oak Ave/)).toBeInTheDocument();
  });

  it("shows line items", async () => {
    render(<ShipmentDetail />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
  });

  it("shows mark shipped button for draft shipment", async () => {
    render(<ShipmentDetail />);
    await waitFor(() => {
      expect(screen.getByText("Mark Shipped")).toBeInTheDocument();
    });
  });

  it("shows pack list button", async () => {
    render(<ShipmentDetail />);
    await waitFor(() => {
      expect(screen.getByText("Pack List")).toBeInTheDocument();
    });
  });

  it("shows not found for missing shipment", async () => {
    mockGetShipment.mockRejectedValueOnce(new Error("not found"));
    render(<ShipmentDetail />);
    await waitFor(() => {
      expect(screen.getByText("Shipment not found")).toBeInTheDocument();
    });
  });
});
