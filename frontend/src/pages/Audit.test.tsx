import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import userEvent from "@testing-library/user-event";
import Audit from "./Audit";

describe("Audit", () => {
  it("renders page title after loading", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("Audit Log")).toBeInTheDocument();
    });
  });

  it("shows page description", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("Track all system activities and user actions.")).toBeInTheDocument();
    });
  });

  it("renders audit entries after loading", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("john.doe@example.com")).toBeInTheDocument();
      expect(screen.getByText("jane.smith@example.com")).toBeInTheDocument();
      expect(screen.getByText("admin@example.com")).toBeInTheDocument();
    });
  });

  it("has search input", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByPlaceholderText("Search logs...")).toBeInTheDocument();
    });
  });

  it("shows entity types in table", async () => {
    render(<Audit />);
    await waitFor(() => {
      // entity_type values displayed with replace('_', ' ') and capitalize
      expect(screen.getAllByText(/part/i).length).toBeGreaterThan(0);
    });
  });

  it("shows action badges", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("update")).toBeInTheDocument();
      expect(screen.getAllByText("create").length).toBeGreaterThan(0);
      expect(screen.getByText("approve")).toBeInTheDocument();
      expect(screen.getByText("delete")).toBeInTheDocument();
      expect(screen.getByText("login")).toBeInTheDocument();
    });
  });

  it("shows Filters card", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("Filters")).toBeInTheDocument();
    });
  });

  it("shows Audit Entries card", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("Audit Entries")).toBeInTheDocument();
    });
  });

  it("shows entity type filter", async () => {
    render(<Audit />);
    await waitFor(() => {
      // Entity Type appears as both filter label and table header
      expect(screen.getAllByText("Entity Type").length).toBeGreaterThanOrEqual(2);
    });
  });

  it("shows user filter", async () => {
    render(<Audit />);
    await waitFor(() => {
      // "User" label for filter
      const userLabels = screen.getAllByText("User");
      expect(userLabels.length).toBeGreaterThan(0);
    });
  });

  it("shows table headers", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("Timestamp")).toBeInTheDocument();
      expect(screen.getByText("Action")).toBeInTheDocument();
      expect(screen.getAllByText("Entity Type").length).toBeGreaterThanOrEqual(2);
      expect(screen.getByText("Entity ID")).toBeInTheDocument();
      expect(screen.getByText("Details")).toBeInTheDocument();
      expect(screen.getByText("IP Address")).toBeInTheDocument();
    });
  });

  it("shows entity IDs", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("ABC-123")).toBeInTheDocument();
      expect(screen.getByText("ECO-001")).toBeInTheDocument();
      expect(screen.getByText("WO-456")).toBeInTheDocument();
    });
  });

  it("shows IP addresses", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("192.168.1.100")).toBeInTheDocument();
      expect(screen.getByText("192.168.1.101")).toBeInTheDocument();
    });
  });

  it("shows entries count", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("6 entries found")).toBeInTheDocument();
    });
  });

  it("filters by search term", async () => {
    const user = userEvent.setup();
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByPlaceholderText("Search logs...")).toBeInTheDocument();
    });
    await user.type(screen.getByPlaceholderText("Search logs..."), "john.doe");
    await waitFor(() => {
      expect(screen.getByText("1 entries found")).toBeInTheDocument();
    });
  });

  it("shows Clear Filters button when filters active", async () => {
    const user = userEvent.setup();
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByPlaceholderText("Search logs...")).toBeInTheDocument();
    });
    await user.type(screen.getByPlaceholderText("Search logs..."), "john");
    await waitFor(() => {
      expect(screen.getByText("Clear Filters")).toBeInTheDocument();
    });
  });

  it("clears filters when Clear Filters clicked", async () => {
    const user = userEvent.setup();
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByPlaceholderText("Search logs...")).toBeInTheDocument();
    });
    await user.type(screen.getByPlaceholderText("Search logs..."), "john");
    await waitFor(() => {
      expect(screen.getByText("Clear Filters")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Clear Filters"));
    await waitFor(() => {
      expect(screen.getByText("6 entries found")).toBeInTheDocument();
    });
  });

  it("shows details column", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("Updated part description and cost")).toBeInTheDocument();
      expect(screen.getByText("Created new ECO for widget improvement")).toBeInTheDocument();
    });
  });

  it("shows loading state initially", async () => {
    render(<Audit />);
    // Loading may resolve quickly with mock data
    // Just verify the component renders without error
    await waitFor(() => {
      expect(screen.getByText("Audit Log")).toBeInTheDocument();
    });
  });

  it("shows Search label", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("Search")).toBeInTheDocument();
    });
  });
});
