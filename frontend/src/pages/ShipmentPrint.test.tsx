import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import { mockShipments, mockPackList } from "../test/mocks";

const mockGetShipment = vi.fn().mockResolvedValue(mockShipments[0]);
const mockGetShipmentPackList = vi.fn().mockResolvedValue(mockPackList);

vi.mock("../lib/api", () => ({
  api: {
    getShipment: (...args: any[]) => mockGetShipment(...args),
    getShipmentPackList: (...args: any[]) => mockGetShipmentPackList(...args),
  },
}));

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useParams: () => ({ id: "SHP-2024-0001" }) };
});

import ShipmentPrint from "./ShipmentPrint";

beforeEach(() => vi.clearAllMocks());

describe("ShipmentPrint", () => {
  it("renders loading state", () => {
    render(<ShipmentPrint />);
    expect(screen.getByText("Loading...")).toBeInTheDocument();
  });

  it("renders pack list after loading", async () => {
    render(<ShipmentPrint />);
    await waitFor(() => {
      expect(screen.getByText("Pack List")).toBeInTheDocument();
    });
    expect(screen.getByText("SHP-2024-0001")).toBeInTheDocument();
    expect(screen.getByText("IPN-001")).toBeInTheDocument();
  });

  it("shows print button", async () => {
    render(<ShipmentPrint />);
    await waitFor(() => {
      expect(screen.getByText("Print Pack List")).toBeInTheDocument();
    });
  });

  it("shows shipment addresses", async () => {
    render(<ShipmentPrint />);
    await waitFor(() => {
      expect(screen.getByText(/123 Main St/)).toBeInTheDocument();
      expect(screen.getByText(/456 Oak Ave/)).toBeInTheDocument();
    });
  });

  it("shows total items count", async () => {
    render(<ShipmentPrint />);
    await waitFor(() => {
      expect(screen.getByText(/Total items: 5/)).toBeInTheDocument();
    });
  });
});
