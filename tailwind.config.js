/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./dist/*/*.html", "./dist/*.html"],
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
