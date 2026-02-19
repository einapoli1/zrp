import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { WebSocketProvider, useWS } from "./WebSocketContext";

// The global mock from setup.ts is active here, so useWebSocket returns mock values

function TestConsumer() {
  const ws = useWS();
  return (
    <div>
      <span data-testid="status">{ws.status}</span>
      <span data-testid="lastEvent">{ws.lastEvent ? JSON.stringify(ws.lastEvent) : "null"}</span>
      <span data-testid="hasSubscribe">{typeof ws.subscribe === "function" ? "yes" : "no"}</span>
    </div>
  );
}

describe("WebSocketContext", () => {
  it("provider renders children", () => {
    render(
      <WebSocketProvider>
        <div data-testid="child">Hello</div>
      </WebSocketProvider>
    );
    expect(screen.getByTestId("child")).toHaveTextContent("Hello");
  });

  it("useWS returns expected shape", () => {
    render(
      <WebSocketProvider>
        <TestConsumer />
      </WebSocketProvider>
    );
    expect(screen.getByTestId("status")).toHaveTextContent("connected");
    expect(screen.getByTestId("lastEvent")).toHaveTextContent("null");
    expect(screen.getByTestId("hasSubscribe")).toHaveTextContent("yes");
  });

  it("useWS throws outside provider", () => {
    // Suppress the expected error
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});
    expect(() => render(<TestConsumer />)).toThrow(
      "useWS must be used within a WebSocketProvider"
    );
    spy.mockRestore();
  });
});
