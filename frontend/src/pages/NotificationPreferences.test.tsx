import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import userEvent from "@testing-library/user-event";

const mockGetTypes = vi.fn();
const mockGetPrefs = vi.fn();
const mockUpdatePrefs = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getNotificationTypes: (...args: any[]) => mockGetTypes(...args),
    getNotificationPreferences: (...args: any[]) => mockGetPrefs(...args),
    updateNotificationPreferences: (...args: any[]) => mockUpdatePrefs(...args),
  },
}));

import NotificationPreferences from "./NotificationPreferences";

const defaultTypes = [
  { type: "low_stock", name: "Low Stock", description: "When inventory drops below threshold", icon: "package", has_threshold: true, threshold_label: "Minimum Qty", threshold_default: 10 },
  { type: "overdue_wo", name: "Overdue Work Order", description: "When a WO is overdue", icon: "clock", has_threshold: true, threshold_label: "Days Overdue", threshold_default: 7 },
  { type: "open_ncr", name: "Open NCR", description: "When NCR is open >14 days", icon: "alert-triangle", has_threshold: false },
  { type: "eco_approval", name: "ECO Approval", description: "ECO approval notifications", icon: "check-circle", has_threshold: false },
];

const defaultPrefs = [
  { id: 1, user_id: 1, notification_type: "low_stock", enabled: true, delivery_method: "in_app", threshold_value: 10 },
  { id: 2, user_id: 1, notification_type: "overdue_wo", enabled: true, delivery_method: "in_app", threshold_value: 7 },
  { id: 3, user_id: 1, notification_type: "open_ncr", enabled: true, delivery_method: "in_app", threshold_value: null },
  { id: 4, user_id: 1, notification_type: "eco_approval", enabled: true, delivery_method: "in_app", threshold_value: null },
];

beforeEach(() => {
  vi.clearAllMocks();
  mockGetTypes.mockResolvedValue([...defaultTypes]);
  mockGetPrefs.mockResolvedValue(defaultPrefs.map(p => ({ ...p })));
  mockUpdatePrefs.mockResolvedValue(defaultPrefs.map(p => ({ ...p })));
});

describe("NotificationPreferences", () => {
  it("renders all notification types", async () => {
    render(<NotificationPreferences />);
    await waitFor(() => {
      expect(screen.getByText("Low Stock")).toBeInTheDocument();
      expect(screen.getByText("Overdue Work Order")).toBeInTheDocument();
      expect(screen.getByText("Open NCR")).toBeInTheDocument();
      expect(screen.getByText("ECO Approval")).toBeInTheDocument();
    });
  });

  it("shows descriptions for each type", async () => {
    render(<NotificationPreferences />);
    await waitFor(() => {
      expect(screen.getByText("When inventory drops below threshold")).toBeInTheDocument();
    });
  });

  it("shows threshold inputs for types with thresholds", async () => {
    render(<NotificationPreferences />);
    await waitFor(() => {
      expect(screen.getByText("Low Stock")).toBeInTheDocument();
    });
    // Should have threshold inputs for low_stock and overdue_wo
    const inputs = screen.getAllByRole("spinbutton");
    expect(inputs.length).toBeGreaterThanOrEqual(2);
  });

  it("calls save with updated preferences", async () => {
    const user = userEvent.setup();
    render(<NotificationPreferences />);
    await waitFor(() => {
      expect(screen.getByText("Low Stock")).toBeInTheDocument();
    });

    const saveButton = screen.getByRole("button", { name: /save/i });
    await user.click(saveButton);
    expect(mockUpdatePrefs).toHaveBeenCalledTimes(1);
  });

  it("has a reset to defaults button", async () => {
    render(<NotificationPreferences />);
    await waitFor(() => {
      expect(screen.getByRole("button", { name: /reset/i })).toBeInTheDocument();
    });
  });

  it("loads preferences on mount", async () => {
    render(<NotificationPreferences />);
    await waitFor(() => {
      expect(mockGetTypes).toHaveBeenCalledTimes(1);
      expect(mockGetPrefs).toHaveBeenCalledTimes(1);
    });
  });
});
