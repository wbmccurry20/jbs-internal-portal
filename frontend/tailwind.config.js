/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{astro,html,js,jsx,md,mdx,svelte,ts,tsx,vue}'],
  theme: {
    extend: {
      colors: {
        primary: {
          DEFAULT: '#1B9BD8',
          50: '#E6F5FB',
          100: '#CCE9F7',
          200: '#99D4EF',
          300: '#66BEE7',
          400: '#33A9DF',
          500: '#1B9BD8',
          600: '#167CAD',
          700: '#105D82',
          800: '#0B3E57',
          900: '#051F2C',
        },
        secondary: '#28a745',
        accent: '#6B7280',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
      },
    },
  },
  plugins: [],
}
