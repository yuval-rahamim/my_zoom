import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import fs from "fs";
import path from "path";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5174,
    https: {
      key: fs.readFileSync(path.resolve("C:/Users/yuval/Documents/GitHub/my_zoom/api/keys", "localhost.key")),
      cert: fs.readFileSync(path.resolve("C:/Users/yuval/Documents/GitHub/my_zoom/api/keys", "localhost.crt"))      
    },
    proxy: {
      "/api": {
        target: "https://localhost:3000",
        changeOrigin: true,
        secure: false // accept self-signed certificates
      }
    }
  }
});
