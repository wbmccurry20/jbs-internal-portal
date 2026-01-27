// API client configuration
const API_BASE_URL = import.meta.env.PUBLIC_API_URL || 'http://localhost:8080/api';
console.log('API_BASE_URL:', API_BASE_URL, 'PUBLIC_API_URL env:', import.meta.env.PUBLIC_API_URL);

export interface LoginResponse {
  token: string;
  user: {
    id: number;
    email: string;
    name: string;
  };
}

export async function login(email: string, password: string): Promise<LoginResponse> {
  const response = await fetch(`${API_BASE_URL}/login`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ email, password }),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Login failed');
  }

  return response.json();
}

export function getAuthToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('auth_token');
}

export function setAuthToken(token: string): void {
  if (typeof window === 'undefined') return;
  localStorage.setItem('auth_token', token);
}

export function clearAuthToken(): void {
  if (typeof window === 'undefined') return;
  localStorage.removeItem('auth_token');
}

export function isAuthenticated(): boolean {
  return !!getAuthToken();
}
