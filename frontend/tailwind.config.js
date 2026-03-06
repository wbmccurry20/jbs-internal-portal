/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{astro,html,js,jsx,md,mdx,svelte,ts,tsx,vue}'],
  theme: {
    extend: {
      colors: {
        primary: {
          DEFAULT: '#00A0E0',
          50:  '#E6F6FD',
          100: '#CCEDFA',
          200: '#99DBF6',
          300: '#66CAF1',
          400: '#33B8EC',
          500: '#00A0E0',
          600: '#0080B3',
          700: '#006086',
          800: '#00405A',
          900: '#00202D',
        },
        secondary: '#28a745',
        accent: '#6B7280',
        // JBS brand slate for nav/sidebar backgrounds
        'jbs-dark': '#1A1A1A',
        'jbs-charcoal': '#3E3832',
      },
      fontFamily: {
        sans: ['Roboto', 'system-ui', 'sans-serif'],
        heading: ['"Barlow Condensed"', 'sans-serif'],
      },
    },
  },
  plugins: [],
}
