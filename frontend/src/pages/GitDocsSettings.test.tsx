import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";

const mockGetGitDocsSettings = vi.fn();
const mockUpdateGitDocsSettings = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getGitDocsSettings: (...args: any[]) => mockGetGitDocsSettings(...args),
    updateGitDocsSettings: (...args: any[]) => mockUpdateGitDocsSettings(...args),
  },
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

import GitDocsSettings from "./GitDocsSettings";

const mockConfig = { repo_url: "https://github.com/org/docs.git", branch: "main", token: "" };

beforeEach(() => {
  vi.clearAllMocks();
  mockGetGitDocsSettings.mockResolvedValue(mockConfig);
  mockUpdateGitDocsSettings.mockResolvedValue({ status: "ok" });
});

describe("GitDocsSettings", () => {
  it("renders loading state initially", () => {
    mockGetGitDocsSettings.mockReturnValue(new Promise(() => {}));
    render(<GitDocsSettings />);
    expect(screen.getByText("Loading...")).toBeInTheDocument();
  });

  it("renders settings form after loading", async () => {
    render(<GitDocsSettings />);
    await waitFor(() => {
      expect(screen.getByText("Git Document Repository")).toBeInTheDocument();
    });
  });

  it("displays existing settings", async () => {
    render(<GitDocsSettings />);
    await waitFor(() => {
      expect(screen.getByDisplayValue("https://github.com/org/docs.git")).toBeInTheDocument();
      expect(screen.getByDisplayValue("main")).toBeInTheDocument();
    });
  });

  it("shows repo URL, branch, and token fields", async () => {
    render(<GitDocsSettings />);
    await waitFor(() => {
      expect(screen.getByLabelText("Repository URL")).toBeInTheDocument();
      expect(screen.getByLabelText("Branch")).toBeInTheDocument();
      expect(screen.getByLabelText("Access Token")).toBeInTheDocument();
    });
  });

  it("saves settings on button click", async () => {
    render(<GitDocsSettings />);
    await waitFor(() => screen.getByText(/save/i));
    fireEvent.click(screen.getByText(/save git docs settings/i));
    await waitFor(() => {
      expect(mockUpdateGitDocsSettings).toHaveBeenCalled();
    });
  });

  it("updates input values when user types", async () => {
    render(<GitDocsSettings />);
    await waitFor(() => screen.getByLabelText("Repository URL"));
    fireEvent.change(screen.getByLabelText("Repository URL"), { target: { value: "https://new-url.git" } });
    expect(screen.getByDisplayValue("https://new-url.git")).toBeInTheDocument();
  });

  it("handles load error gracefully", async () => {
    mockGetGitDocsSettings.mockRejectedValueOnce(new Error("fail"));
    render(<GitDocsSettings />);
    await waitFor(() => {
      expect(screen.getByText("Git Document Repository")).toBeInTheDocument();
    });
  });
});
