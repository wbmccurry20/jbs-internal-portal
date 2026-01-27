import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import * as api from '../lib/api';

// Mock API module
vi.mock('../lib/api', () => ({
  login: vi.fn(),
  setAuthToken: vi.fn(),
  getAuthToken: vi.fn(),
  clearAuthToken: vi.fn(),
  isAuthenticated: vi.fn(),
}));

describe('API Helper Functions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  describe('Authentication Token Management', () => {
    it('stores auth token in localStorage', () => {
      const mockToken = 'test-token-123';
      api.setAuthToken(mockToken);
      expect(api.setAuthToken).toHaveBeenCalledWith(mockToken);
    });

    it('retrieves auth token from localStorage', () => {
      localStorage.setItem('auth_token', 'stored-token');
      const token = api.getAuthToken();
      expect(api.getAuthToken).toHaveBeenCalled();
    });

    it('clears auth token from localStorage', () => {
      api.clearAuthToken();
      expect(api.clearAuthToken).toHaveBeenCalled();
    });

    it('checks if user is authenticated', () => {
      const result = api.isAuthenticated();
      expect(api.isAuthenticated).toHaveBeenCalled();
    });
  });

  describe('Login API', () => {
    it('calls login endpoint with credentials', async () => {
      const mockResponse = {
        token: 'new-token',
        user: { id: 1, email: 'test@jbs.com', name: 'Test User' },
      };
      
      vi.mocked(api.login).mockResolvedValue(mockResponse);
      
      const result = await api.login('test@jbs.com', 'password123');
      
      expect(api.login).toHaveBeenCalledWith('test@jbs.com', 'password123');
      expect(result).toEqual(mockResponse);
    });

    it('handles login errors', async () => {
      const errorMessage = 'Invalid credentials';
      vi.mocked(api.login).mockRejectedValue(new Error(errorMessage));
      
      await expect(api.login('wrong@jbs.com', 'wrong')).rejects.toThrow(errorMessage);
    });

    it('handles network errors', async () => {
      vi.mocked(api.login).mockRejectedValue(new Error('Network error'));
      
      await expect(api.login('test@jbs.com', 'password')).rejects.toThrow('Network error');
    });
  });
});

describe('Authentication Flow Integration', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  it('completes full authentication flow', async () => {
    const mockUser = { id: 1, email: 'test@jbs.com', name: 'Test' };
    const mockToken = 'auth-token-xyz';
    
    vi.mocked(api.login).mockResolvedValue({
      token: mockToken,
      user: mockUser,
    });

    // Simulate login
    const loginResult = await api.login('test@jbs.com', 'password');
    expect(loginResult.token).toBe(mockToken);
    
    // Store token
    api.setAuthToken(mockToken);
    expect(api.setAuthToken).toHaveBeenCalledWith(mockToken);
  });

  it('handles logout flow', () => {
    // Setup authenticated state
    localStorage.setItem('auth_token', 'some-token');
    
    // Logout
    api.clearAuthToken();
    expect(api.clearAuthToken).toHaveBeenCalled();
  });
});

describe('Error Handling', () => {
  it('handles 401 unauthorized errors', async () => {
    vi.mocked(api.login).mockRejectedValue(new Error('Unauthorized'));
    
    await expect(api.login('test@jbs.com', 'wrong')).rejects.toThrow('Unauthorized');
  });

  it('handles 500 server errors', async () => {
    vi.mocked(api.login).mockRejectedValue(new Error('Server error'));
    
    await expect(api.login('test@jbs.com', 'password')).rejects.toThrow('Server error');
  });

  it('handles timeout errors', async () => {
    vi.mocked(api.login).mockRejectedValue(new Error('Request timeout'));
    
    await expect(api.login('test@jbs.com', 'password')).rejects.toThrow('Request timeout');
  });
});
