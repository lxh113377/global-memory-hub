/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        gold: "#d4a017",
        flame: "#ff6b1a",
      },
      boxShadow: {
        glow: "0 0 18px rgba(255, 107, 26, 0.45)",
      },
      keyframes: {
        breath: {
          "0%, 100%": { opacity: "1", boxShadow: "0 0 5px 1px currentColor" },
          "50%": { opacity: "0.5", boxShadow: "0 0 11px 3px currentColor" },
        },
      },
      animation: {
        breath: "breath 2.4s ease-in-out infinite",
      },
    },
  },
  plugins: [],
};
