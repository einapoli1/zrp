import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";

const mockGetGitPLMConfig = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getGitPLMConfig: (...args: any[]) => mockGetGitPLMConfig(...args),
  },
}));

import { useGitPLM } from "./useGitPLM";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("useGitPLM", () => {
  it("returns configured=false when no URL is set", async () => {
    mockGetGitPLMConfig.mockResolvedValue({ base_url: "" });
    const { result } = renderHook(() => useGitPLM());
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.configured).toBe(false);
    expect(result.current.buildUrl("IPN-001")).toBeNull();
  });

  it("returns configured=true and builds URLs when configured", async () => {
    mockGetGitPLMConfig.mockResolvedValue({ base_url: "https://plm.acme.com" });
    const { result } = renderHook(() => useGitPLM());
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.configured).toBe(true);
    expect(result.current.buildUrl("IPN-001")).toBe("https://plm.acme.com/parts/IPN-001");
  });

  it("handles API errors gracefully", async () => {
    mockGetGitPLMConfig.mockRejectedValue(new Error("fail"));
    const { result } = renderHook(() => useGitPLM());
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.configured).toBe(false);
  });
});
