// API client configuration
const API_BASE_URL = import.meta.env.PUBLIC_API_URL || 'http://localhost:8080/api';

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
  return localStorage.getItem('jbs_token');
}

export function setAuthToken(token: string): void {
  if (typeof window === 'undefined') return;
  localStorage.setItem('jbs_token', token);
}

export function clearAuthToken(): void {
  if (typeof window === 'undefined') return;
  localStorage.removeItem('jbs_token');
}

export function isAuthenticated(): boolean {
  return !!getAuthToken();
}

// User interface
export interface User {
  id: number;
  email: string;
  name: string;
  role: string;
  created_at: string;
}

// User Management API
export async function listUsers(): Promise<User[]> {
  const token = getAuthToken();
  const response = await fetch(`${API_BASE_URL}/users`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to fetch users');
  }

  const data = await response.json();
  return data.users;
}

export async function createUser(email: string, password: string, name: string, role: string): Promise<void> {
  const token = getAuthToken();
  const response = await fetch(`${API_BASE_URL}/users`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ email, password, name, role }),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to create user');
  }
}

export async function deleteUser(userId: number): Promise<void> {
  const token = getAuthToken();
  const response = await fetch(`${API_BASE_URL}/users/${userId}`, {
    method: 'DELETE',
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to delete user');
  }
}

export async function resetUserPassword(userId: number, newPassword: string): Promise<void> {
  const token = getAuthToken();
  const response = await fetch(`${API_BASE_URL}/users/${userId}/reset-password`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ new_password: newPassword }),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to reset password');
  }
}

// Change own password
export async function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  const token = getAuthToken();
  const response = await fetch(`${API_BASE_URL}/change-password`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to change password');
  }
}

// Invite a new user (sends email with invite link)
export async function inviteUser(email: string, name: string, role: string): Promise<{ message: string; user_id: number }> {
  const token = getAuthToken();
  const response = await fetch(`${API_BASE_URL}/users/invite`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ email, name, role }),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to invite user');
  }

  return response.json();
}

// Resend invite email for a pending user
export async function resendInvite(userId: number): Promise<void> {
  const token = getAuthToken();
  const response = await fetch(`${API_BASE_URL}/users/${userId}/resend-invite`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to resend invite');
  }
}

// Validate an invite token (public — no auth needed)
export async function validateInviteToken(inviteToken: string): Promise<{ valid: boolean; name?: string; email?: string }> {
  const response = await fetch(`${API_BASE_URL}/invite/validate/${inviteToken}`);

  if (!response.ok) {
    return { valid: false };
  }

  return response.json();
}

// Accept an invite and set password (public — no auth needed)
export async function acceptInvite(inviteToken: string, password: string): Promise<{ message: string }> {
  const response = await fetch(`${API_BASE_URL}/invite/accept`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ token: inviteToken, password }),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to accept invite');
  }

  return response.json();
}
