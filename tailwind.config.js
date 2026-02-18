/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: 'class',
  content: [
    './templates/**/*.html',
    './static/**/*.js',
    './static/**/*.html',
  ],
  theme: {
    extend: {
      borderWidth: {
        '3': '3px',
      }
    },
  },
  plugins: [],
}
