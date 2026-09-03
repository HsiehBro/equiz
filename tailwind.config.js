/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./pages/**/*.{wxml,js,ts}", "./components/**/*.{wxml,js,ts}", "./app.wxml"],
  plugins: [require("@tailwindcss/forms"), require("@tailwindcss/container-queries")], theme: {
    extend: {
      colors: {
        "surface": "#f8f9fa",
        "surface-dim": "#d9dadb",
        "surface-bright": "#f8f9fa",
        "surface-container-lowest": "#ffffff",
        "surface-container-low": "#f3f4f5",
        "surface-container": "#edeeef",
        "surface-container-high": "#e7e8e9",
        "surface-container-highest": "#e1e3e4",
        "on-surface": "#191c1d",
        "on-surface-variant": "#414755",
        "inverse-surface": "#2e3132",
        "inverse-on-surface": "#f0f1f2",
        "outline": "#717786",
        "outline-variant": "#c1c6d7",
        "surface-tint": "#005bc1",
        "primary": "#0058bc",
        "on-primary": "#ffffff",
        "primary-container": "#0070eb",
        "on-primary-container": "#fefcff",
        "inverse-primary": "#adc6ff",
        "secondary": "#006e28",
        "on-secondary": "#ffffff",
        "secondary-container": "#6ffb85",
        "on-secondary-container": "#00732a",
        "tertiary": "#bc000a",
        "on-tertiary": "#ffffff",
        "tertiary-container": "#e2241f",
        "on-tertiary-container": "#fffbff",
        "error": "#ba1a1a",
        "on-error": "#ffffff",
        "error-container": "#ffdad6",
        "on-error-container": "#93000a",
        "background": "#f8f9fa",
        "on-background": "#191c1d"
      },
      fontFamily: {
        "inter": ["Inter", "sans-serif"]
      },
      fontSize: {
        "display": ["30px", { "lineHeight": "38px", "letterSpacing": "-0.5px", "fontWeight": "700" }],
        "headline-lg": ["22px", { "lineHeight": "28px", "fontWeight": "600" }],
        "headline-md": ["18px", { "lineHeight": "24px", "fontWeight": "600" }],
        "body-lg": ["17px", { "lineHeight": "26px", "fontWeight": "400" }],
        "body-md": ["15px", { "lineHeight": "22px", "fontWeight": "400" }],
        "label-md": ["13px", { "lineHeight": "18px", "letterSpacing": "0.2px", "fontWeight": "500" }],
        "label-sm": ["11px", { "lineHeight": "16px", "fontWeight": "600" }]
      },
      spacing: {
        "base": "4px",
        "xs": "8px",
        "sm": "12px",
        "md": "16px",
        "lg": "24px",
        "xl": "32px",
        "margin-mobile": "16px",
        "gutter-card": "12px"
      },
      borderRadius: {
        "sm": "0.25rem",
        "DEFAULT": "0.5rem",
        "md": "0.75rem",
        "lg": "1rem",
        "xl": "1.5rem",
        "full": "9999px"
      },
      boxShadow: {
        "card": "0 2px 8px rgba(0, 0, 0, 0.05)",
        "modal": "0 8px 24px rgba(0, 0, 0, 0.1)"
      }
    }
  }
}
