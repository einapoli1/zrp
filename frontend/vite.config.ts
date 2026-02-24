import path from "path"
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import { visualizer } from 'rollup-plugin-visualizer'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    visualizer({
      filename: './dist/stats.html',
      open: false,
      gzipSize: true,
      brotliSize: true,
    })
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: (id) => {
          // Node modules chunking - avoid circular dependencies by being specific
          if (id.includes('node_modules')) {
            // Barcode/QR code libraries (heavy) - isolate completely
            if (id.includes('html5-qrcode') || id.includes('quagga') || id.includes('@zxing')) {
              return 'barcode-libs';
            }
            // QR code generation (separate from scanner)
            if (id.includes('qrcode.react')) {
              return 'qrcode-gen';
            }
            // Command palette
            if (id.includes('cmdk')) {
              return 'cmdk';
            }
            // Core React - be very specific to avoid pulling in other deps
            if (id.includes('node_modules/react/') && !id.includes('react-dom') && !id.includes('react-router')) {
              return 'react-core';
            }
            if (id.includes('node_modules/react-dom/')) {
              return 'react-core';
            }
            if (id.includes('scheduler')) {
              return 'react-core';
            }
            // React Router
            if (id.includes('react-router')) {
              return 'react-router';
            }
            // Radix UI - keep together to avoid circular deps
            if (id.includes('@radix-ui')) {
              return 'radix-ui';
            }
            // Form libraries
            if (id.includes('react-hook-form') || id.includes('zod') || id.includes('@hookform')) {
              return 'form-libs';
            }
            // Icons
            if (id.includes('lucide-react')) {
              return 'lucide';
            }
            // Toast notifications
            if (id.includes('sonner')) {
              return 'toast';
            }
            // Utilities (lightweight)
            if (id.includes('clsx') || id.includes('tailwind-merge') || id.includes('class-variance-authority')) {
              return 'utils';
            }
            // Don't create a vendor chunk - let remaining modules stay in entry
          }
          
          // App code chunking by feature
          // UI components
          if (id.includes('/components/ui/')) {
            return 'ui-components';
          }
          // Common components
          if (id.includes('/components/')) {
            return 'components';
          }
          // API client
          if (id.includes('/lib/api')) {
            return 'api-client';
          }
          // Contexts
          if (id.includes('/contexts/')) {
            return 'contexts';
          }
        },
      },
    },
    // Increase chunk size warning limit
    chunkSizeWarningLimit: 600,
  },
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:9000",
        changeOrigin: true,
      },
      "/auth": {
        target: "http://localhost:9000",
        changeOrigin: true,
      },
      "/files": {
        target: "http://localhost:9000",
        changeOrigin: true,
      },
    },
  },
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: ["./src/test/setup.ts"],
    include: ["src/**/*.test.{ts,tsx}"],
    css: true,
  },
})
