import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// https://vitejs.dev/config/
// base 用相对路径: 同一份产物既可被本地 agent 托管在 / 也可部署在 Pages 的 /console/ 子路径。
export default defineConfig({
  base: "./",
  plugins: [react()],
});
