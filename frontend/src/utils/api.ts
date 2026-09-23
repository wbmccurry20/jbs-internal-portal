/**
 * Centralized API configuration.
 * All pages should use these helpers instead of constructing API URLs directly.
 */

function normalizeApiBase(rawUrl?: string): string {
  if (!rawUrl) return '';

  const trimmed = rawUrl.trim();
  if (!trimmed) return '';

  const withoutTrailingSlash = trimmed.replace(/\/+$/, '');
  if (/\/api$/i.test(withoutTrailingSlash)) {
    return withoutTrailingSlash;
  }

  return `${withoutTrailingSlash}/api`;
}

/**
 * Returns the base API URL (ending in /api) for standard REST endpoints.
 * Uses PUBLIC_API_URL env var in production, falls back to localhost only in dev.
 */
export function getApiUrl(): string {
  const fromEnv = normalizeApiBase(import.meta.env.PUBLIC_API_URL);
  return fromEnv || (import.meta.env.DEV ? 'http://localhost:8080/api' : '');
}

/**
 * Returns the base server URL (no /api suffix) for endpoints like /api/concur/upload
 * that need the root server path.
 */
export function getServerUrl(): string {
  const fromEnv = normalizeApiBase(import.meta.env.PUBLIC_API_URL);
  return fromEnv ? fromEnv.replace(/\/api$/, '') : (import.meta.env.DEV ? 'http://localhost:8080' : '');
}
