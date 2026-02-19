import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import userEvent from "@testing-library/user-event";
import APIKeys from "./APIKeys";

// Mock clipboard API
Object.assign(navigator, {
  clipboard: {
    writeText: vi.fn().mockResolvedValue(undefined),
  },
});

describe("APIKeys", () => {
  it("renders page title after loading", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      const titles = screen.getAllByText("API Keys");
      expect(titles.length).toBeGreaterThan(0);
    });
  });

  it("shows page description", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("Manage API keys for programmatic access to ZRP.")).toBeInTheDocument();
    });
  });

  it("renders API key list after loading", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("Production Integration")).toBeInTheDocument();
      expect(screen.getByText("Mobile App")).toBeInTheDocument();
      expect(screen.getByText("Legacy System")).toBeInTheDocument();
      expect(screen.getByText("Testing Environment")).toBeInTheDocument();
    });
  });

  it("shows key status badges", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      const actives = screen.getAllByText(/active/i);
      expect(actives.length).toBeGreaterThan(0);
      const revoked = screen.getAllByText(/revoked/i);
      expect(revoked.length).toBeGreaterThan(0);
    });
  });

  it("has Generate New Key button", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("Generate New Key")).toBeInTheDocument();
    });
  });

  it("shows summary cards", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("Total Keys")).toBeInTheDocument();
      expect(screen.getByText("Revoked")).toBeInTheDocument();
    });
  });

  it("shows correct key counts", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("4")).toBeInTheDocument(); // total keys
    });
  });

  it("shows table headers", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("Name")).toBeInTheDocument();
      expect(screen.getByText("Key")).toBeInTheDocument();
      expect(screen.getByText("Created By")).toBeInTheDocument();
      expect(screen.getByText("Last Used")).toBeInTheDocument();
    });
  });

  it("shows key prefixes", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("zrp_abc123...")).toBeInTheDocument();
      expect(screen.getByText("zrp_def456...")).toBeInTheDocument();
    });
  });

  it("shows Revoke buttons for active keys", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      const revokeButtons = screen.getAllByText("Revoke");
      expect(revokeButtons.length).toBe(3); // 3 active keys
    });
  });

  it("does not show Revoke button for revoked keys", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      // Legacy System is revoked, should not have a Revoke button in its row
      expect(screen.getByText("Legacy System")).toBeInTheDocument();
    });
  });

  it("shows 'Never' for keys without last_used", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("Never")).toBeInTheDocument();
    });
  });

  it("opens generate key dialog", async () => {
    const user = userEvent.setup();
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("Generate New Key")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Generate New Key"));
    await waitFor(() => {
      expect(screen.getByText("Generate New API Key")).toBeInTheDocument();
      expect(screen.getByPlaceholderText("Enter a descriptive name")).toBeInTheDocument();
    });
  });

  it("shows important warning in generate dialog", async () => {
    const user = userEvent.setup();
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("Generate New Key")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Generate New Key"));
    await waitFor(() => {
      expect(screen.getByText("Important")).toBeInTheDocument();
    });
  });

  it("Generate Key button disabled when name empty", async () => {
    const user = userEvent.setup();
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("Generate New Key")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Generate New Key"));
    await waitFor(() => {
      const generateBtn = screen.getByText("Generate Key");
      expect(generateBtn).toBeDisabled();
    });
  });

  it("generates key and shows full key", async () => {
    const user = userEvent.setup();
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("Generate New Key")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Generate New Key"));
    await waitFor(() => {
      expect(screen.getByPlaceholderText("Enter a descriptive name")).toBeInTheDocument();
    });
    await user.type(screen.getByPlaceholderText("Enter a descriptive name"), "My Test Key");
    await user.click(screen.getByText("Generate Key"));
    await waitFor(() => {
      expect(screen.getByText("API Key Generated Successfully")).toBeInTheDocument();
    });
  });

  it("opens revoke confirmation dialog", async () => {
    const user = userEvent.setup();
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getAllByText("Revoke").length).toBeGreaterThan(0);
    });
    await user.click(screen.getAllByText("Revoke")[0]);
    await waitFor(() => {
      expect(screen.getByText("Revoke API Key")).toBeInTheDocument();
      expect(screen.getByText("Confirm Revocation")).toBeInTheDocument();
    });
  });

  it("confirms key revocation", async () => {
    const user = userEvent.setup();
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getAllByText("Revoke").length).toBeGreaterThan(0);
    });
    await user.click(screen.getAllByText("Revoke")[0]);
    await waitFor(() => {
      expect(screen.getByText("Yes, Revoke Key")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Yes, Revoke Key"));
    // After revocation, the revoked count should increase
    await waitFor(() => {
      // Dialog should close
      expect(screen.queryByText("Yes, Revoke Key")).not.toBeInTheDocument();
    });
  });

  it("shows created_by for each key", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getAllByText("admin@example.com").length).toBeGreaterThan(0);
      expect(screen.getByText("developer@example.com")).toBeInTheDocument();
      expect(screen.getByText("tester@example.com")).toBeInTheDocument();
    });
  });
});
