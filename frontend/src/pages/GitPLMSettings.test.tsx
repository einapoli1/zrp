import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";

const mockGetGitPLMConfig = vi.fn();
const mockUpdateGitPLMConfig = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getGitPLMConfig: (...args: any[]) => mockGetGitPLMConfig(...args),
    updateGitPLMConfig: (...args: any[]) => mockUpdateGitPLMConfig(...args),
  },
}));

import GitPLMSettings from "./GitPLMSettings";

beforeEach(() => {
  vi.clearAllMocks();
  mockGetGitPLMConfig.mockResolvedValue({ base_url: "" });
  mockUpdateGitPLMConfig.mockResolvedValue({ base_url: "" });
});

describe("GitPLMSettings", () => {
  it("renders the settings page", async () => {
    render(<GitPLMSettings />);
    await waitFor(() => expect(screen.getByText("GitPLM Integration")).toBeInTheDocument());
    expect(screen.getByLabelText("Base URL")).toBeInTheDocument();
  });

  it("shows 'Not configured' when no URL is set", async () => {
    render(<GitPLMSettings />);
    await waitFor(() => expect(screen.getByText("Not configured")).toBeInTheDocument());
  });

  it("shows 'Configured' when URL is set", async () => {
    mockGetGitPLMConfig.mockResolvedValue({ base_url: "https://gitplm.example.com" });
    render(<GitPLMSettings />);
    await waitFor(() => expect(screen.getByText("Configured")).toBeInTheDocument());
  });

  it("saves the URL when Save is clicked", async () => {
    mockUpdateGitPLMConfig.mockResolvedValue({ base_url: "https://gitplm.example.com" });
    render(<GitPLMSettings />);
    await waitFor(() => expect(screen.getByLabelText("Base URL")).toBeInTheDocument());

    fireEvent.change(screen.getByLabelText("Base URL"), {
      target: { value: "https://gitplm.example.com" },
    });
    fireEvent.click(screen.getByText("Save"));

    await waitFor(() =>
      expect(mockUpdateGitPLMConfig).toHaveBeenCalledWith({ base_url: "https://gitplm.example.com" })
    );
  });

  it("shows success message after save", async () => {
    mockUpdateGitPLMConfig.mockResolvedValue({ base_url: "https://gitplm.example.com" });
    render(<GitPLMSettings />);
    await waitFor(() => expect(screen.getByLabelText("Base URL")).toBeInTheDocument());

    fireEvent.change(screen.getByLabelText("Base URL"), {
      target: { value: "https://gitplm.example.com" },
    });
    fireEvent.click(screen.getByText("Save"));

    await waitFor(() => expect(screen.getByText("Saved successfully")).toBeInTheDocument());
  });

  it("disables test connection when no URL", async () => {
    render(<GitPLMSettings />);
    await waitFor(() => expect(screen.getByText("Test Connection")).toBeDisabled());
  });
});
