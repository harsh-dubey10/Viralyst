/** @type {import('tailwindcss').Config} */
// Palette and type borrowed from Shopify Editions: warm parchment / olive-ink neutrals,
// a tight bold grotesk for display, a serif for accents. Colours are CSS variables
// (see index.css) so light and dark share one set of class names.
const v = (name) => `rgb(var(--${name}) / <alpha-value>)`;

module.exports = {
  darkMode: "class",
  content: ["./src/**/*.{js,jsx,ts,tsx}"],
  theme: {
    extend: {
      colors: {
        page: v("page"),
        surface: v("surface"),
        raised: v("raised"),
        ink: v("ink"),
        accent: v("accent"),
        "accent-strong": v("accent-strong"),
        good: v("good"),
        warn: v("warn"),
        bad: v("bad"),
      },
      fontFamily: {
        sans: ["Inter", "ui-sans-serif", "system-ui", "sans-serif"],
        display: ['"Inter Tight"', "Inter", "ui-sans-serif", "system-ui", "sans-serif"],
        serif: ['"Instrument Serif"', "ui-serif", "Georgia", "serif"],
      },
      transitionTimingFunction: {
        spring: "cubic-bezier(.34,1.56,.64,1)",
      },
      boxShadow: {
        card: "3px 6px 12px rgba(0,0,0,.13)",
        glow: "0 0 64px rgb(var(--accent) / .35)",
      },
      keyframes: {
        rise: {
          from: { transform: "translateY(105%)", opacity: "0" },
          to: { transform: "translateY(0)", opacity: "1" },
        },
        "fade-up": {
          from: { transform: "translateY(16px)", opacity: "0" },
          to: { transform: "translateY(0)", opacity: "1" },
        },
        "nav-in": {
          from: { transform: "translateY(-100%)", opacity: "0" },
          to: { transform: "translateY(0)", opacity: "1" },
        },
        grow: {
          from: { transform: "scaleX(0)" },
          to: { transform: "scaleX(1)" },
        },
        shimmer: {
          from: { backgroundPosition: "200% 0" },
          to: { backgroundPosition: "-200% 0" },
        },
      },
      animation: {
        rise: "rise .9s cubic-bezier(.34,1.56,.64,1) both",
        "fade-up": "fade-up .7s cubic-bezier(.22,1,.36,1) both",
        "nav-in": "nav-in .6s cubic-bezier(.34,1.56,.64,1) both",
        shimmer: "shimmer 2.4s linear infinite",
      },
    },
  },
  plugins: [],
};
