import path from "path";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
// Relative base so embedded Wails WebView2 reliably resolves /assets chunks (avoids blank UI when absolute paths fail).
export default defineConfig(({ command }) => ({
  base: command === "build" ? "./" : "/",
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    // WebView2 / современные Chromium: без легаси-транспиля под IE
    target: "es2022",
    chunkSizeWarningLimit: 600,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes("node_modules")) return;
          // Крупные библиотеки — отдельные чанки для кэширования и параллельной загрузки
          if (id.includes("skinview3d") || id.includes("/three/") || id.includes("\\three\\")) {
            return "skinview3d";
          }
          if (id.includes("@radix-ui")) {
            return "radix-ui";
          }
          if (id.includes("lucide-react")) {
            return "lucide";
          }
          if (id.includes("@tabler/icons-react")) {
            return "tabler-icons";
          }
          if (id.includes("react-dom") || id.includes("/react/") || id.includes("\\react\\")) {
            return "react-vendor";
          }
        },
      },
    },
  },
}));
