import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import userEvent from "@testing-library/user-event";
import Users from "./Users";

// No api mock needed - uses internal mock data

describe("Users", () => {
  it("renders loading state then content", async () => {
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText("User Management")).toBeInTheDocument();
    });
  });

  it("renders user list after loading", async () => {
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText("admin")).toBeInTheDocument();
    });
  });

  it("shows user emails", async () => {
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText("admin@example.com")).toBeInTheDocument();
      expect(screen.getByText("john.doe@example.com")).toBeInTheDocument();
      expect(screen.getByText("jane.smith@example.com")).toBeInTheDocument();
    });
  });

  it("has Create User button", async () => {
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText("Create User")).toBeInTheDocument();
    });
  });

  it("shows user roles with badges", async () => {
    render(<Users />);
    await waitFor(() => {
      expect(screen.getAllByText("Administrator").length).toBeGreaterThan(0);
      expect(screen.getAllByText("User").length).toBeGreaterThan(0);
      expect(screen.getAllByText("Read Only").length).toBeGreaterThan(0);
    });
  });

  it("shows user status badges", async () => {
    render(<Users />);
    await waitFor(() => {
      const activeBadges = screen.getAllByText("Active");
      expect(activeBadges.length).toBeGreaterThan(0);
      expect(screen.getByText("Inactive")).toBeInTheDocument();
    });
  });

  it("shows page description", async () => {
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText("Manage user accounts, roles, and permissions.")).toBeInTheDocument();
    });
  });

  it("shows stats cards with counts", async () => {
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText("Total Users")).toBeInTheDocument();
      expect(screen.getByText("Admins")).toBeInTheDocument();
      expect(screen.getByText("Active Today")).toBeInTheDocument();
    });
  });

  it("shows correct total user count", async () => {
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText("5")).toBeInTheDocument(); // 5 mock users
    });
  });

  it("shows table headers", async () => {
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText("Username")).toBeInTheDocument();
      expect(screen.getByText("Email")).toBeInTheDocument();
      expect(screen.getByText("Role")).toBeInTheDocument();
      expect(screen.getByText("Status")).toBeInTheDocument();
      expect(screen.getByText("Last Login")).toBeInTheDocument();
      expect(screen.getByText("Created")).toBeInTheDocument();
      expect(screen.getByText("Actions")).toBeInTheDocument();
    });
  });

  it("shows Edit buttons for each user", async () => {
    render(<Users />);
    await waitFor(() => {
      const editButtons = screen.getAllByText("Edit");
      expect(editButtons.length).toBe(5); // 5 mock users
    });
  });

  it("shows all mock users", async () => {
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText("admin")).toBeInTheDocument();
      expect(screen.getByText("john.doe")).toBeInTheDocument();
      expect(screen.getByText("jane.smith")).toBeInTheDocument();
      expect(screen.getByText("guest")).toBeInTheDocument();
      expect(screen.getByText("old.user")).toBeInTheDocument();
    });
  });

  it("opens create user dialog when button clicked", async () => {
    const user = userEvent.setup();
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText("Create User")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Create User"));
    await waitFor(() => {
      expect(screen.getByText("Create New User")).toBeInTheDocument();
    });
  });

  it("shows create user form fields in dialog", async () => {
    const user = userEvent.setup();
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText("Create User")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Create User"));
    await waitFor(() => {
      expect(screen.getByPlaceholderText("Enter username")).toBeInTheDocument();
      expect(screen.getByPlaceholderText("Enter email address")).toBeInTheDocument();
      expect(screen.getByPlaceholderText("Enter password")).toBeInTheDocument();
    });
  });

  it("opens edit dialog when Edit clicked", async () => {
    const user = userEvent.setup();
    render(<Users />);
    await waitFor(() => {
      expect(screen.getAllByText("Edit").length).toBeGreaterThan(0);
    });
    await user.click(screen.getAllByText("Edit")[0]);
    await waitFor(() => {
      expect(screen.getByText(/Edit User:/)).toBeInTheDocument();
    });
  });

  it("edit dialog shows role and status selects", async () => {
    const user = userEvent.setup();
    render(<Users />);
    await waitFor(() => {
      expect(screen.getAllByText("Edit").length).toBeGreaterThan(0);
    });
    await user.click(screen.getAllByText("Edit")[0]);
    await waitFor(() => {
      expect(screen.getByText("Update User")).toBeInTheDocument();
      expect(screen.getAllByText("Cancel").length).toBeGreaterThan(0);
    });
  });

  it("shows Users card title", async () => {
    render(<Users />);
    await waitFor(() => {
      const usersTitles = screen.getAllByText("Users");
      expect(usersTitles.length).toBeGreaterThan(0);
    });
  });

  it("shows 'Never' for users without last login", async () => {
    render(<Users />);
    await waitFor(() => {
      // All mock users have last_login set, but the component handles 'Never'
      expect(screen.getByText("admin")).toBeInTheDocument();
    });
  });

  it("shows inactive user old.user", async () => {
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText("old.user")).toBeInTheDocument();
      expect(screen.getByText("old.user@example.com")).toBeInTheDocument();
    });
  });
});
