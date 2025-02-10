import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5174, // Set frontend port to 5174
    proxy: {
      "/api": {
        target: "http://localhost:3000", // Proxy backend API requests
        changeOrigin: true,
        secure: false
      }
    }
  }
});
