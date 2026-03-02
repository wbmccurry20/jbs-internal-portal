/**
 * Centralized API configuration.
 * All pages should use these helpers instead of constructing API URLs directly.
 */

/**
 * Returns the base API URL (ending in /api) for standard REST endpoints.
 * Uses PUBLIC_API_URL env var in production, falls back to localhost only in dev.
 */
export function getApiUrl(): string {
  return import.meta.env.PUBLIC_API_URL || (import.meta.env.DEV ? 'http://localhost:8080/api' : '');
}

/**
 * Returns the base server URL (no /api suffix) for endpoints like /api/concur/upload
 * that need the root server path.
 */
export function getServerUrl(): string {
  return import.meta.env.PUBLIC_API_URL?.replace(/\/api$/, '') || (import.meta.env.DEV ? 'http://localhost:8080' : '');
}
