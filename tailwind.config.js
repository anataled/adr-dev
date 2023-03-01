/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./base/*.html", "./partials/*.html"],
  theme: {
    fontFamily: {
      sans: ['"Merriweather Sans"', 'sans-serif'],
      serif: ['"Merriweather"', 'serif'],
      mono: ['"Cutive Mono"', 'serif']
    },
    extend: {
    },
  },
  plugins: [],
}
