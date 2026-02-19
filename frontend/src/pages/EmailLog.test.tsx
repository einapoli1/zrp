import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";

const mockGetLog = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getEmailLog: (...args: any[]) => mockGetLog(...args),
  },
}));

import EmailLog from "./EmailLog";

const mockEntries = [
  {
    id: 1,
    to_address: "user@test.com",
    subject: "ECO-001 Approved",
    body: "Your ECO was approved",
    event_type: "eco_approved",
    status: "sent",
    error: "",
    sent_at: "2024-01-15 10:30:00",
  },
  {
    id: 2,
    to_address: "admin@test.com",
    subject: "Low Stock: IPN-001",
    body: "Low stock alert",
    event_type: "low_stock",
    status: "failed",
    error: "SMTP timeout",
    sent_at: "2024-01-15 11:00:00",
  },
];

beforeEach(() => {
  vi.clearAllMocks();
  mockGetLog.mockResolvedValue(mockEntries);
});

describe("EmailLog", () => {
  it("renders page title", async () => {
    render(<EmailLog />);
    await waitFor(() => {
      expect(screen.getByText("Email Log")).toBeInTheDocument();
    });
  });

  it("shows email entries", async () => {
    render(<EmailLog />);
    await waitFor(() => {
      expect(screen.getByText("user@test.com")).toBeInTheDocument();
      expect(screen.getByText("ECO-001 Approved")).toBeInTheDocument();
      expect(screen.getByText("eco_approved")).toBeInTheDocument();
    });
  });

  it("shows failed status with error", async () => {
    render(<EmailLog />);
    await waitFor(() => {
      expect(screen.getByText("SMTP timeout")).toBeInTheDocument();
      expect(screen.getByText("failed")).toBeInTheDocument();
    });
  });

  it("shows sent count in header", async () => {
    render(<EmailLog />);
    await waitFor(() => {
      expect(screen.getByText("Sent Emails (2)")).toBeInTheDocument();
    });
  });

  it("shows empty state when no emails", async () => {
    mockGetLog.mockResolvedValue([]);
    render(<EmailLog />);
    await waitFor(() => {
      expect(screen.getByText(/No emails sent yet/i)).toBeInTheDocument();
    });
  });

  it("shows table headers", async () => {
    render(<EmailLog />);
    await waitFor(() => {
      expect(screen.getByText("To")).toBeInTheDocument();
      expect(screen.getByText("Subject")).toBeInTheDocument();
      expect(screen.getByText("Event")).toBeInTheDocument();
      expect(screen.getByText("Status")).toBeInTheDocument();
      expect(screen.getByText("Sent At")).toBeInTheDocument();
    });
  });
});
