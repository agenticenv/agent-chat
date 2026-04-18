import { reactRouter } from "@react-router/dev/vite"
import tailwindcss from "@tailwindcss/vite"
import { defineConfig, loadEnv } from "vite"

export default defineConfig(({ mode }) => {
  const fileEnv = loadEnv(mode, process.cwd(), "")
  const serverApiUrl =
    process.env.SERVER_API_URL ?? fileEnv.SERVER_API_URL ?? "http://localhost:8080"

  return {
    plugins: [tailwindcss(), reactRouter()],
    resolve: {
      dedupe: ["react", "react-dom"],
    },
    server: {
      proxy: {
        "/api": {
          target: serverApiUrl,
          changeOrigin: true,
        },
      },
    },
  }
})
