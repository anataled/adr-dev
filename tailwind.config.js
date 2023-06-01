/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./base/*.html", "./partials/*.html", "./base/locations/*.html"],
  theme: {
    fontFamily: {
      sans: ['Bahnschrift', '"DIN Alternate"', '"Franklin Gothic Medium"', '"Nimbus Sans Narrow"', 'sans-serif-condensed', 'sans-serif'],
      serif: ['Poppins', 'serif'],
      mono: ["ui-monospace", '"Cascadia Code"', '"Source Code Pro"', 'Menlo', 'Consolas', '"DejaVu Sans Mono"', 'monospace']
    },
    extend: {
    },
  },
  plugins: [],
}
