import { defineConfig } from 'astro/config';
import tailwind from '@astrojs/tailwind';
import node from '@astrojs/node';

export default defineConfig({
  output: 'server',
  adapter: node({
    mode: 'standalone',
    host: '0.0.0.0', // Listen on all interfaces for Railway
    port: 8080
  }),
  integrations: [tailwind()],
  server: {
    port: 4321,
    host: true // Allow external connections
  },
  vite: {
    server: {
      allowedHosts: [
        'welcoming-rejoicing-production.up.railway.app',
        'portal.jbsconstructiongroup.com'
      ]
    }
  }
});
