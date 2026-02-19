import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import userEvent from "@testing-library/user-event";

const mockGetSubs = vi.fn();
const mockUpdateSubs = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getEmailSubscriptions: (...args: any[]) => mockGetSubs(...args),
    updateEmailSubscriptions: (...args: any[]) => mockUpdateSubs(...args),
  },
}));

import EmailPreferences from "./EmailPreferences";

const defaultSubs = {
  eco_approved: true,
  eco_implemented: true,
  low_stock: true,
  overdue_work_order: true,
  po_received: true,
  ncr_created: true,
};

beforeEach(() => {
  vi.clearAllMocks();
  mockGetSubs.mockResolvedValue({ ...defaultSubs });
  mockUpdateSubs.mockResolvedValue({ ...defaultSubs, eco_approved: false });
});

describe("EmailPreferences", () => {
  it("renders page title", async () => {
    render(<EmailPreferences />);
    await waitFor(() => {
      expect(screen.getByText("Email Preferences")).toBeInTheDocument();
    });
  });

  it("shows all event type checkboxes", async () => {
    render(<EmailPreferences />);
    await waitFor(() => {
      expect(screen.getByText(/ECO Approved/)).toBeInTheDocument();
      expect(screen.getByText(/ECO Implemented/)).toBeInTheDocument();
      expect(screen.getByText(/Low Stock Alert/)).toBeInTheDocument();
      expect(screen.getByText(/Overdue Work Order/)).toBeInTheDocument();
      expect(screen.getByText(/PO Received/)).toBeInTheDocument();
      expect(screen.getByText(/NCR Created/)).toBeInTheDocument();
    });
  });

  it("save button is disabled when no changes", async () => {
    render(<EmailPreferences />);
    await waitFor(() => {
      const btn = screen.getByText("Save Preferences").closest("button");
      expect(btn).toBeDisabled();
    });
  });

  it("save button enables after toggling a checkbox", async () => {
    const user = userEvent.setup();
    render(<EmailPreferences />);
    await waitFor(() => {
      expect(screen.getByText(/ECO Approved/)).toBeInTheDocument();
    });
    await user.click(screen.getByText(/ECO Approved/));
    await waitFor(() => {
      const btn = screen.getByText("Save Preferences").closest("button");
      expect(btn).not.toBeDisabled();
    });
  });

  it("calls updateEmailSubscriptions on save", async () => {
    const user = userEvent.setup();
    render(<EmailPreferences />);
    await waitFor(() => {
      expect(screen.getByText(/ECO Approved/)).toBeInTheDocument();
    });
    await user.click(screen.getByText(/ECO Approved/));
    await user.click(screen.getByText("Save Preferences"));
    await waitFor(() => {
      expect(mockUpdateSubs).toHaveBeenCalled();
    });
  });

  it("shows description text", async () => {
    render(<EmailPreferences />);
    await waitFor(() => {
      expect(screen.getByText(/Choose which email notifications/)).toBeInTheDocument();
    });
  });
});
