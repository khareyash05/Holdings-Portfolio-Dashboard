import animate from "tailwindcss-animate";

/** @type {import('tailwindcss').Config} */
export default {
  darkMode: ["class"],
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        border: "hsl(0 0% 16%)",
        input: "hsl(0 0% 12%)",
        ring: "hsl(0 0% 60%)",
        background: "hsl(0 0% 5.5%)",
        foreground: "hsl(0 0% 90%)",
        primary: {
          DEFAULT: "hsl(0 0% 90%)",
          foreground: "hsl(0 0% 9%)",
        },
        secondary: {
          DEFAULT: "hsl(0 0% 9%)",
          foreground: "hsl(0 0% 90%)",
        },
        destructive: {
          DEFAULT: "hsl(0 60% 60%)",
          foreground: "hsl(0 0% 98%)",
        },
        muted: {
          DEFAULT: "hsl(0 0% 9%)",
          foreground: "hsl(0 0% 53%)",
        },
        accent: {
          DEFAULT: "hsl(0 0% 9%)",
          foreground: "hsl(0 0% 90%)",
        },
        popover: {
          DEFAULT: "hsl(0 0% 5.5%)",
          foreground: "hsl(0 0% 90%)",
        },
        card: {
          DEFAULT: "hsl(0 0% 5.5%)",
          foreground: "hsl(0 0% 90%)",
        },
        pos: "hsl(138 35% 55%)",
        neg: "hsl(0 65% 67%)",
      },
      borderRadius: {
        lg: "0.25rem",
        md: "calc(0.25rem - 2px)",
        sm: "calc(0.25rem - 4px)",
      },
      fontFamily: {
        sans: [
          "-apple-system",
          "BlinkMacSystemFont",
          '"Segoe UI"',
          "system-ui",
          "sans-serif",
        ],
        mono: ["ui-monospace", "SFMono-Regular", "Menlo", "monospace"],
      },
      keyframes: {
        shimmer: {
          "0%": { backgroundPosition: "200% 0" },
          "100%": { backgroundPosition: "-200% 0" },
        },
      },
      animation: {
        shimmer: "shimmer 1.2s infinite linear",
      },
    },
  },
  plugins: [animate],
};
