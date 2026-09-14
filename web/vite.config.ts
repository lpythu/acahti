import path from "node:path"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

const origin = process.env.ACAHTI_DEV_ORIGIN || "http://192.168.0.180:8080"

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "./src"),
    },
  },
  server: {
    host: "127.0.0.1",
    port: 5173,
    proxy: {
      "/ui": {
        target: origin,
        changeOrigin: true,
        cookieDomainRewrite: "",
      },
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
})
