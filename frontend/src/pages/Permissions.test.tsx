import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";

const mockModules = [
  { module: "parts", actions: ["view", "create", "edit", "delete"] },
  { module: "ecos", actions: ["view", "create", "edit", "delete", "approve"] },
];
const mockPerms = [
  { module: "parts", action: "view" },
  { module: "parts", action: "create" },
  { module: "ecos", action: "view" },
];
const mockGetPermissions = vi.fn().mockResolvedValue(mockPerms);
const mockGetPermissionModules = vi.fn().mockResolvedValue(mockModules);
const mockSetRolePermissions = vi.fn().mockResolvedValue(undefined);

vi.mock("../lib/api", () => ({
  getPermissions: (...args: any[]) => mockGetPermissions(...args),
  getPermissionModules: (...args: any[]) => mockGetPermissionModules(...args),
  setRolePermissions: (...args: any[]) => mockSetRolePermissions(...args),
}));

// Mock sonner toast
vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

import Permissions from "./Permissions";

beforeEach(() => vi.clearAllMocks());

describe("Permissions", () => {
  it("renders permissions page", async () => {
    render(<Permissions />);
    await waitFor(() => {
      expect(mockGetPermissionModules).toHaveBeenCalled();
    });
  });

  it("loads permissions for default role", async () => {
    render(<Permissions />);
    await waitFor(() => {
      expect(mockGetPermissions).toHaveBeenCalledWith("admin");
    });
  });

  it("shows module names", async () => {
    render(<Permissions />);
    await waitFor(() => {
      expect(screen.getByText("Parts")).toBeInTheDocument();
      expect(screen.getByText("ECOs")).toBeInTheDocument();
    });
  });

  it("shows save button", async () => {
    render(<Permissions />);
    await waitFor(() => {
      expect(screen.getByText(/save/i)).toBeInTheDocument();
    });
  });

  it("handles load error gracefully", async () => {
    mockGetPermissionModules.mockRejectedValueOnce(new Error("fail"));
    render(<Permissions />);
    // Should not crash
    await waitFor(() => {
      expect(mockGetPermissionModules).toHaveBeenCalled();
    });
  });
});
