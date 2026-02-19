import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "../test/test-utils";
import Login from "./Login";

// Mock react-router-dom navigate
const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

// Mock the api module
vi.mock("../lib/api", () => ({
  api: {
    login: vi.fn(),
  },
}));

import { api } from "../lib/api";

describe("Login Page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders login form with username, password, and submit button", () => {
    render(<Login />);
    expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /sign in/i })).toBeInTheDocument();
  });

  it("submits credentials and redirects to dashboard on success", async () => {
    (api.login as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      user: { id: 1, username: "admin", display_name: "Admin", role: "admin" },
    });

    render(<Login />);
    fireEvent.change(screen.getByLabelText(/username/i), { target: { value: "admin" } });
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: "changeme" } });
    fireEvent.click(screen.getByRole("button", { name: /sign in/i }));

    await waitFor(() => {
      expect(api.login).toHaveBeenCalledWith("admin", "changeme");
      expect(mockNavigate).toHaveBeenCalledWith("/dashboard");
    });
  });

  it("displays error message on login failure", async () => {
    (api.login as ReturnType<typeof vi.fn>).mockRejectedValueOnce(
      new Error("Invalid credentials")
    );

    render(<Login />);
    fireEvent.change(screen.getByLabelText(/username/i), { target: { value: "admin" } });
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: "wrong" } });
    fireEvent.click(screen.getByRole("button", { name: /sign in/i }));

    await waitFor(() => {
      expect(screen.getByText(/invalid credentials/i)).toBeInTheDocument();
    });
  });

  it("shows loading state while submitting", async () => {
    let resolveLogin: (value: any) => void;
    (api.login as ReturnType<typeof vi.fn>).mockReturnValueOnce(
      new Promise((resolve) => { resolveLogin = resolve; })
    );

    render(<Login />);
    fireEvent.change(screen.getByLabelText(/username/i), { target: { value: "admin" } });
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: "pass" } });
    fireEvent.click(screen.getByRole("button", { name: /sign in/i }));

    await waitFor(() => {
      expect(screen.getByRole("button", { name: /signing in/i })).toBeDisabled();
    });

    resolveLogin!({ user: { id: 1 } });
  });
});
