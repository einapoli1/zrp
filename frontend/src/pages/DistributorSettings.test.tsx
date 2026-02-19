import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";

const mockGetDistributorSettings = vi.fn().mockResolvedValue({
  digikey: { client_id: "dk-id", client_secret: "dk-secret" },
  mouser: { api_key: "mouser-key" },
});
const mockUpdateDigikeySettings = vi.fn().mockResolvedValue({ status: "ok" });
const mockUpdateMouserSettings = vi.fn().mockResolvedValue({ status: "ok" });

vi.mock("../lib/api", () => ({
  api: {
    getDistributorSettings: (...args: any[]) => mockGetDistributorSettings(...args),
    updateDigikeySettings: (...args: any[]) => mockUpdateDigikeySettings(...args),
    updateMouserSettings: (...args: any[]) => mockUpdateMouserSettings(...args),
  },
}));

import DistributorSettings from "./DistributorSettings";

beforeEach(() => vi.clearAllMocks());

describe("DistributorSettings", () => {
  it("renders page title after loading", async () => {
    render(<DistributorSettings />);
    await waitFor(() => {
      expect(screen.getByText("Distributor API Settings")).toBeInTheDocument();
    });
  });

  it("loads and displays existing settings", async () => {
    render(<DistributorSettings />);
    await waitFor(() => {
      expect(screen.getByDisplayValue("dk-id")).toBeInTheDocument();
    });
  });

  it("shows Digikey and Mouser sections", async () => {
    render(<DistributorSettings />);
    await waitFor(() => {
      expect(screen.getByText("Digikey")).toBeInTheDocument();
      expect(screen.getByText("Mouser")).toBeInTheDocument();
    });
  });

  it("saves Digikey settings", async () => {
    render(<DistributorSettings />);
    await waitFor(() => screen.getByText("Digikey"));
    const saveButtons = screen.getAllByText(/save/i);
    fireEvent.click(saveButtons[0]);
    await waitFor(() => {
      expect(mockUpdateDigikeySettings).toHaveBeenCalled();
    });
  });

  it("shows success message after save", async () => {
    render(<DistributorSettings />);
    await waitFor(() => screen.getByText("Digikey"));
    const saveButtons = screen.getAllByText(/save/i);
    fireEvent.click(saveButtons[0]);
    await waitFor(() => {
      expect(screen.getByText("Digikey settings saved")).toBeInTheDocument();
    });
  });

  it("shows error message on save failure", async () => {
    mockUpdateDigikeySettings.mockRejectedValueOnce(new Error("Failed to save Digikey settings"));
    render(<DistributorSettings />);
    await waitFor(() => screen.getByText("Digikey"));
    const saveButtons = screen.getAllByText(/save/i);
    fireEvent.click(saveButtons[0]);
    await waitFor(() => {
      expect(screen.getByText("Failed to save Digikey settings")).toBeInTheDocument();
    });
  });

  it("handles empty settings gracefully", async () => {
    mockGetDistributorSettings.mockResolvedValueOnce({});
    render(<DistributorSettings />);
    await waitFor(() => {
      expect(screen.getByText("Distributor API Settings")).toBeInTheDocument();
    });
  });
});
