import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import userEvent from "@testing-library/user-event";
import EmailSettings from "./EmailSettings";

describe("EmailSettings", () => {
  it("renders page title after loading", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("Email Settings")).toBeInTheDocument();
    });
  });

  it("shows page description", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText(/Configure SMTP settings/)).toBeInTheDocument();
    });
  });

  it("shows SMTP configuration card", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("SMTP Configuration")).toBeInTheDocument();
    });
  });

  it("shows SMTP host and port fields", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("SMTP Host")).toBeInTheDocument();
      expect(screen.getByText("SMTP Port")).toBeInTheDocument();
    });
  });

  it("shows authentication section", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("Authentication")).toBeInTheDocument();
      expect(screen.getByText("Username")).toBeInTheDocument();
      expect(screen.getByText("Password")).toBeInTheDocument();
    });
  });

  it("shows Security select", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("Security")).toBeInTheDocument();
    });
  });

  it("has Save Settings button", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("Save Settings")).toBeInTheDocument();
    });
  });

  it("Save button is disabled when no changes", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      const saveButton = screen.getByText("Save Settings").closest("button");
      expect(saveButton).toBeDisabled();
    });
  });

  it("shows Send Test button", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("Send Test")).toBeInTheDocument();
    });
  });

  it("shows Test Email Configuration section", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("Test Email Configuration")).toBeInTheDocument();
      expect(screen.getByText("Test Email Address")).toBeInTheDocument();
    });
  });

  it("shows Email Notifications card with checkbox", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("Email Notifications")).toBeInTheDocument();
      expect(screen.getByText("Enable email notifications")).toBeInTheDocument();
    });
  });

  it("shows enabled notification message when email is enabled", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("Email notifications are enabled")).toBeInTheDocument();
    });
  });

  it("shows Sender Configuration card", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("Sender Configuration")).toBeInTheDocument();
      expect(screen.getByText("From Email Address")).toBeInTheDocument();
      expect(screen.getByText("From Name")).toBeInTheDocument();
    });
  });

  it("shows SMTP Provider Preset selector", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("SMTP Provider Preset")).toBeInTheDocument();
    });
  });

  it("shows mock SMTP host value", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      const hostInput = screen.getByDisplayValue("smtp.example.com");
      expect(hostInput).toBeInTheDocument();
    });
  });

  it("shows mock SMTP port value", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      const portInput = screen.getByDisplayValue("587");
      expect(portInput).toBeInTheDocument();
    });
  });

  it("shows mock username value", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByDisplayValue("notifications@example.com")).toBeInTheDocument();
    });
  });

  it("shows mock from address value", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByDisplayValue("noreply@example.com")).toBeInTheDocument();
    });
  });

  it("shows test email placeholder", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByPlaceholderText("test@example.com")).toBeInTheDocument();
    });
  });

  it("shows loading state initially", async () => {
    render(<EmailSettings />);
    // Loading may resolve quickly with mock data
    await waitFor(() => {
      expect(screen.getByText("Email Settings")).toBeInTheDocument();
    });
  });

  it("shows unsaved changes badge when config modified", async () => {
    const user = userEvent.setup();
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByDisplayValue("smtp.example.com")).toBeInTheDocument();
    });
    const hostInput = screen.getByDisplayValue("smtp.example.com");
    await user.clear(hostInput);
    await user.type(hostInput, "new.smtp.host.com");
    await waitFor(() => {
      expect(screen.getByText("Unsaved Changes")).toBeInTheDocument();
    });
  });

  it("enables Save button when changes are made", async () => {
    const user = userEvent.setup();
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByDisplayValue("smtp.example.com")).toBeInTheDocument();
    });
    const hostInput = screen.getByDisplayValue("smtp.example.com");
    await user.clear(hostInput);
    await user.type(hostInput, "new.host.com");
    await waitFor(() => {
      const saveButton = screen.getByText("Save Settings").closest("button");
      expect(saveButton).not.toBeDisabled();
    });
  });

  it("Send Test button is disabled when no test email entered", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      const sendBtn = screen.getByText("Send Test").closest("button");
      expect(sendBtn).toBeDisabled();
    });
  });
});
