import { defineConfig } from 'astro/config';
import react from '@astrojs/react';
import tailwind from '@astrojs/tailwind';
import node from '@astrojs/node';

export default defineConfig({
  output: 'server',
  adapter: node({
    mode: 'standalone'
  }),
  integrations: [react(), tailwind()],
  server: {
    port: 4321,
    host: true, // Allow external connections
    headers: {
      'Content-Security-Policy': "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; connect-src 'self' http://localhost:8080 https://fonts.googleapis.com; img-src 'self' data:;"
    }
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
