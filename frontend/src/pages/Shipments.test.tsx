import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockShipments } from "../test/mocks";

const mockGetShipments = vi.fn().mockResolvedValue(mockShipments);
const mockCreateShipment = vi.fn().mockResolvedValue(mockShipments[0]);

vi.mock("../lib/api", () => ({
  api: {
    getShipments: (...args: any[]) => mockGetShipments(...args),
    createShipment: (...args: any[]) => mockCreateShipment(...args),
  },
}));

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useNavigate: () => mockNavigate };
});

import Shipments from "./Shipments";

beforeEach(() => vi.clearAllMocks());

describe("Shipments", () => {
  it("renders loading state", () => {
    render(<Shipments />);
    expect(screen.getByText("Loading shipments...")).toBeInTheDocument();
  });

  it("renders shipment list after loading", async () => {
    render(<Shipments />);
    await waitFor(() => {
      expect(screen.getByText("SHP-2024-0001")).toBeInTheDocument();
    });
    expect(screen.getByText("SHP-2024-0002")).toBeInTheDocument();
  });

  it("has create shipment button", async () => {
    render(<Shipments />);
    await waitFor(() => {
      expect(screen.getByText("Create Shipment")).toBeInTheDocument();
    });
  });

  it("shows empty state", async () => {
    mockGetShipments.mockResolvedValueOnce([]);
    render(<Shipments />);
    await waitFor(() => {
      expect(screen.getByText(/No shipments found/)).toBeInTheDocument();
    });
  });

  it("shows carrier info", async () => {
    render(<Shipments />);
    await waitFor(() => {
      expect(screen.getByText("FedEx")).toBeInTheDocument();
      expect(screen.getByText("UPS")).toBeInTheDocument();
    });
  });

  it("shows tracking number", async () => {
    render(<Shipments />);
    await waitFor(() => {
      expect(screen.getByText("1Z999")).toBeInTheDocument();
    });
  });

  it("navigates to detail on row click", async () => {
    render(<Shipments />);
    await waitFor(() => {
      expect(screen.getByText("SHP-2024-0001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("SHP-2024-0001"));
    expect(mockNavigate).toHaveBeenCalledWith("/shipments/SHP-2024-0001");
  });
});
