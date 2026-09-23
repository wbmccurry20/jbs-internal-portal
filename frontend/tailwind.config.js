/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{astro,html,js,jsx,md,mdx,svelte,ts,tsx,vue}'],
  theme: {
    extend: {
      colors: {
        primary: {
          DEFAULT: '#00A0E0',
          50: '#E6F6FD',
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
        'jbs-blue': '#00A0E0',
        'jbs-blue-hover': '#0088CC',
        'jbs-dark': '#1A1A1A',
        'jbs-black': '#000000',
        'jbs-charcoal': '#3E3832',
        'jbs-gray': '#B8B8B8',
        'jbs-cream': '#F0E8E0',
        'jbs-beige': '#D8D0C8',
        'jbs-brown': '#885830',
        'jbs-sage': '#788078',
        'jbs-gold': '#C0A870',
        'jbs-light-blue': '#C0D8F0',
        'jbs-white': '#FFFFFF',
        'jbs-canvas': '#F7F6F4',
      },
      fontFamily: {
        sans: ['Roboto', 'system-ui', 'sans-serif'],
        heading: ['"Barlow Condensed"', 'sans-serif'],
      },
      boxShadow: {
        card: '0 1px 2px rgba(0, 0, 0, 0.04)',
      },
    },
  },
  plugins: [],
}
