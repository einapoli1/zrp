import React from "react";
import { render, type RenderOptions } from "@testing-library/react";
import { BrowserRouter } from "react-router-dom";
import { Toaster } from "sonner";
import { WebSocketProvider } from "../contexts/WebSocketContext";
import { PermissionsProvider } from "../contexts/PermissionsContext";

function AllProviders({ children }: { children: React.ReactNode }) {
  return (
    <BrowserRouter>
      <WebSocketProvider>
        <PermissionsProvider>
          {children}
          <Toaster />
        </PermissionsProvider>
      </WebSocketProvider>
    </BrowserRouter>
  );
}

const customRender = (
  ui: React.ReactElement,
  options?: Omit<RenderOptions, "wrapper">
) => render(ui, { wrapper: AllProviders, ...options });

export * from "@testing-library/react";
export { customRender as render };

// Re-export accessibility testing utilities
export { expectNoA11yViolations, testA11y, checkA11y, formatA11yViolations } from "./a11y-test-utils";
