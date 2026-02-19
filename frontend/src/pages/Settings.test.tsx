import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import userEvent from "@testing-library/user-event";

const mockGetGeneralSettings = vi.fn();
const mockUpdateGeneralSettings = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getGeneralSettings: (...args: any[]) => mockGetGeneralSettings(...args),
    updateGeneralSettings: (...args: any[]) => mockUpdateGeneralSettings(...args),
    getEmailConfig: vi.fn().mockResolvedValue({ smtp_host: "", smtp_port: 587, from_address: "", from_name: "", smtp_user: "", smtp_pass: "", enabled: false }),
    updateEmailConfig: vi.fn().mockResolvedValue({}),
    testEmail: vi.fn().mockResolvedValue({ success: true }),
    getGitPLMConfig: vi.fn().mockResolvedValue({ base_url: "" }),
    updateGitPLMConfig: vi.fn().mockResolvedValue({}),
    getDistributorSettings: vi.fn().mockResolvedValue({ digikey_configured: false, mouser_configured: false }),
    updateDigikeySettings: vi.fn().mockResolvedValue({}),
    updateMouserSettings: vi.fn().mockResolvedValue({}),
    getBackups: vi.fn().mockResolvedValue([]),
    createBackup: vi.fn().mockResolvedValue({}),
    getUsers: vi.fn().mockResolvedValue([]),
  },
}));

import Settings from "./Settings";

beforeEach(() => {
  vi.clearAllMocks();
  mockGetGeneralSettings.mockResolvedValue({
    app_name: "ZRP",
    company_name: "",
    company_address: "",
    currency: "USD",
    date_format: "YYYY-MM-DD",
  });
  mockUpdateGeneralSettings.mockResolvedValue({});
});

describe("Settings", () => {
  it("renders settings page title", async () => {
    render(<Settings />);
    expect(screen.getByText("Settings")).toBeInTheDocument();
  });

  it("shows tabbed navigation", async () => {
    render(<Settings />);
    expect(screen.getByRole("tab", { name: /general/i })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /email/i })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /distributor/i })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /gitplm/i })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /backup/i })).toBeInTheDocument();
  });

  it("loads general settings on mount", async () => {
    render(<Settings />);
    await waitFor(() => {
      expect(mockGetGeneralSettings).toHaveBeenCalled();
    });
  });

  it("shows general settings form fields", async () => {
    render(<Settings />);
    await waitFor(() => {
      expect(screen.getByLabelText(/app name/i)).toBeInTheDocument();
    });
    expect(screen.getByLabelText(/company name/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/currency/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/date format/i)).toBeInTheDocument();
  });

  it("can save general settings", async () => {
    const user = userEvent.setup();
    render(<Settings />);
    await waitFor(() => {
      expect(screen.getByLabelText(/app name/i)).toBeInTheDocument();
    });

    const appNameInput = screen.getByLabelText(/app name/i);
    await user.clear(appNameInput);
    await user.type(appNameInput, "MyApp");

    const saveBtn = screen.getByRole("button", { name: /save/i });
    await user.click(saveBtn);

    await waitFor(() => {
      expect(mockUpdateGeneralSettings).toHaveBeenCalled();
    });
  });
});
