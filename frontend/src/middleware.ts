import { defineMiddleware } from 'astro:middleware';

export const onRequest = defineMiddleware(async (context, next) => {
  const response = await next();

  // Security headers
  response.headers.set('X-Frame-Options', 'DENY');
  response.headers.set('X-Content-Type-Options', 'nosniff');
  response.headers.set('Referrer-Policy', 'strict-origin-when-cross-origin');
  response.headers.set('Permissions-Policy', 'geolocation=(), microphone=(), camera=()');
  // Build CSP with environment-aware backend URL (origin only, no path)
  const isDev = import.meta.env.DEV;
  const rawUrl = isDev 
    ? 'http://localhost:8080' 
    : (import.meta.env.PUBLIC_API_URL || 'https://jbs-internal-portal-production.up.railway.app');
  let backendUrl = rawUrl;
  try {
    const parsed = new URL(rawUrl);
    backendUrl = `${parsed.protocol}//${parsed.host}`;
  } catch { /* use rawUrl as-is */ }
  
  response.headers.set(
    'Content-Security-Policy',
    `default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; img-src 'self' data: https:; font-src 'self' data: https://fonts.gstatic.com; connect-src 'self' ${backendUrl}; frame-ancestors 'none';`
  );
  response.headers.set(
    'Strict-Transport-Security',
    'max-age=31536000; includeSubDomains; preload'
  );
  // X-XSS-Protection: 0 is recommended by OWASP when CSP is present
  response.headers.set('X-XSS-Protection', '0');

  return response;
});
