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
        manualChunks: {
          // Separate React and React DOM into their own chunk
          'react-vendor': ['react', 'react-dom', 'react/jsx-runtime'],
          // Router into separate chunk
          'react-router': ['react-router-dom'],
          // Radix UI components (large dependency)
          'radix-ui': [
            '@radix-ui/react-avatar',
            '@radix-ui/react-checkbox',
            '@radix-ui/react-dialog',
            '@radix-ui/react-dropdown-menu',
            '@radix-ui/react-label',
            '@radix-ui/react-select',
            '@radix-ui/react-separator',
            '@radix-ui/react-slot',
            '@radix-ui/react-tabs',
            '@radix-ui/react-tooltip',
          ],
          // Form libraries
          'form-libs': ['react-hook-form', '@hookform/resolvers', 'zod'],
          // Icons (can be large)
          'lucide': ['lucide-react'],
          // Other utilities
          'utils': ['clsx', 'tailwind-merge', 'class-variance-authority'],
        },
      },
    },
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
