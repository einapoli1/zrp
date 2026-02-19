import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

// We need to unmock useWebSocket for this test file since setup.ts mocks it globally
vi.unmock("../hooks/useWebSocket");

// Mock WebSocket
class MockWebSocket {
  static instances: MockWebSocket[] = [];
  url: string;
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onmessage: ((evt: { data: string }) => void) | null = null;
  onerror: (() => void) | null = null;
  readyState = 0;

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
    // Simulate async open
    setTimeout(() => {
      this.readyState = 1;
      this.onopen?.();
    }, 10);
  }

  close() {
    this.readyState = 3;
    this.onclose?.();
  }

  send(_data: string) {}

  // Test helper to simulate incoming message
  simulateMessage(data: any) {
    this.onmessage?.({ data: JSON.stringify(data) });
  }
}

describe("useWebSocket", () => {
  beforeEach(() => {
    MockWebSocket.instances = [];
    vi.stubGlobal("WebSocket", MockWebSocket);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("connects and reports connected status", async () => {
    const { useWebSocket } = await import("../hooks/useWebSocket");
    const { result } = renderHook(() => useWebSocket({ reconnect: false }));

    // Initially connecting
    expect(result.current.status).toBe("connecting");

    // Wait for connection
    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(result.current.status).toBe("connected");
  });

  it("reconnects on disconnect", async () => {
    const { useWebSocket } = await import("../hooks/useWebSocket");
    const { result } = renderHook(() => useWebSocket({ reconnect: true, maxDelay: 100 }));

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(result.current.status).toBe("connected");
    const initialCount = MockWebSocket.instances.length;

    // Simulate disconnect
    act(() => {
      MockWebSocket.instances[0].close();
    });

    expect(result.current.status).toBe("disconnected");

    // Wait for reconnect
    await act(async () => {
      await new Promise((r) => setTimeout(r, 1200));
    });

    expect(MockWebSocket.instances.length).toBeGreaterThan(initialCount);
  });

  it("delivers events via subscribe", async () => {
    const { useWebSocket } = await import("../hooks/useWebSocket");
    const { result } = renderHook(() => useWebSocket({ reconnect: false }));

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    const received: any[] = [];
    act(() => {
      result.current.subscribe("eco_updated", (evt) => received.push(evt));
    });

    // Simulate incoming message
    act(() => {
      MockWebSocket.instances[0].simulateMessage({
        type: "eco_updated",
        id: 1,
        action: "update",
      });
    });

    expect(received).toHaveLength(1);
    expect(received[0].type).toBe("eco_updated");
  });

  it("unsubscribe stops delivery", async () => {
    const { useWebSocket } = await import("../hooks/useWebSocket");
    const { result } = renderHook(() => useWebSocket({ reconnect: false }));

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    const received: any[] = [];
    let unsub: () => void;
    act(() => {
      unsub = result.current.subscribe("eco_updated", (evt) => received.push(evt));
    });

    act(() => {
      unsub();
    });

    act(() => {
      MockWebSocket.instances[0].simulateMessage({
        type: "eco_updated",
        id: 1,
        action: "update",
      });
    });

    expect(received).toHaveLength(0);
  });
});
