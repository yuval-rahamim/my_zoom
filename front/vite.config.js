import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import fs from "fs";
import path from "path";

export default defineConfig({
  plugins: [react()],
  server: {
    host: 'myzoom.co.il',
    port: 5174,
    https: {
      key: fs.readFileSync(path.resolve("C:/Users/yuval/Documents/GitHub/my_zoom/api/keys", "myzoom.key")),
      cert: fs.readFileSync(path.resolve("C:/Users/yuval/Documents/GitHub/my_zoom/api/keys", "myzoom.crt"))      
    },
    proxy: {
      "/api": {
        target: "https://myzoom.co.il:3000",
        changeOrigin: true,
        secure: true
      }
    }
  }
});
