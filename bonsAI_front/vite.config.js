import { defineConfig, loadEnv } from "vite";
import preact from "@preact/preset-vite";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const apiTarget = env.BONSAI_DEV_API_TARGET || "http://127.0.0.1:8082";

  return {
    plugins: [preact()],
    base: "./",
    server: {
      host: "0.0.0.0",
      proxy: {
        "/api": {
          target: apiTarget,
          changeOrigin: true
        },
        "/healthz": {
          target: apiTarget,
          changeOrigin: true
        }
      }
    },
    build: {
      outDir: "dist",
      emptyOutDir: true,
      target: "es2020"
    }
  };
});
