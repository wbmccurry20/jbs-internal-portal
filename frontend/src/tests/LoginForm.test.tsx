import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import LoginForm from '../components/LoginForm';
import * as api from '../lib/api';

// Mock the API module
vi.mock('../lib/api', () => ({
  login: vi.fn(),
  setAuthToken: vi.fn(),
  getAuthToken: vi.fn(),
  clearAuthToken: vi.fn(),
  isAuthenticated: vi.fn(),
}));

describe('LoginForm - JBS Portal', () => {
  // Store original location
  const originalLocation = window.location;

  beforeEach(() => {
    vi.clearAllMocks();
    // Mock window.location
    delete (window as any).location;
    (window as any).location = { href: '' };
  });

  afterEach(() => {
    // Restore original location
    (window as any).location = originalLocation;
  });

  it('renders login form with JBS branding', () => {
    render(<LoginForm />);
    
    expect(screen.getByText('JBS Internal Portal')).toBeInTheDocument();
    expect(screen.getByText('Sign in to access tools and reports')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Email address')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Password')).toBeInTheDocument();
  });

  it('submits form with valid credentials', async () => {
    const user = userEvent.setup();
    const mockLogin = vi.spyOn(api, 'login').mockResolvedValue({
      token: 'mock-token',
      user: { id: 1, email: 'test@jbs.com', name: 'Test User' },
    });

    render(<LoginForm />);
    
    const emailInput = screen.getByPlaceholderText('Email address');
    const passwordInput = screen.getByPlaceholderText('Password');
    const submitButton = screen.getByRole('button', { name: /sign in/i });
    
    await user.type(emailInput, 'emily.simpson@jbsconstructiongroup.com');
    await user.type(passwordInput, 'password123');
    await user.click(submitButton);
    
    expect(mockLogin).toHaveBeenCalledWith('emily.simpson@jbsconstructiongroup.com', 'password123');
    expect(api.setAuthToken).toHaveBeenCalledWith('mock-token');
    
    await waitFor(() => {
      expect(window.location.href).toBe('/dashboard');
    });
  });

  it('displays error message on failed login', async () => {
    const user = userEvent.setup();
    const mockLogin = vi.spyOn(api, 'login').mockRejectedValue(
      new Error('Invalid credentials')
    );

    render(<LoginForm />);
    
    const emailInput = screen.getByPlaceholderText('Email address');
    const passwordInput = screen.getByPlaceholderText('Password');
    const submitButton = screen.getByRole('button', { name: /sign in/i });
    
    await user.type(emailInput, 'wrong@jbs.com');
    await user.type(passwordInput, 'wrongpassword');
    await user.click(submitButton);
    
    await waitFor(() => {
      expect(screen.getByText('Invalid credentials')).toBeInTheDocument();
    });
  });

  it('shows loading state during submission', async () => {
    const user = userEvent.setup();
    vi.spyOn(api, 'login').mockImplementation(
      () => new Promise(resolve => setTimeout(resolve, 100))
    );

    render(<LoginForm />);
    
    const emailInput = screen.getByPlaceholderText('Email address');
    const passwordInput = screen.getByPlaceholderText('Password');
    const submitButton = screen.getByRole('button', { name: /sign in/i });
    
    await user.type(emailInput, 'test@jbs.com');
    await user.type(passwordInput, 'password123');
    await user.click(submitButton);
    
    expect(screen.getByRole('button', { name: /signing in/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /signing in/i })).toBeDisabled();
  });

  it('requires email and password fields', () => {
    render(<LoginForm />);
    
    const emailInput = screen.getByPlaceholderText('Email address');
    const passwordInput = screen.getByPlaceholderText('Password');
    
    expect(emailInput).toBeRequired();
    expect(passwordInput).toBeRequired();
  });

  it('uses correct input types for security', () => {
    render(<LoginForm />);
    
    const emailInput = screen.getByPlaceholderText('Email address');
    const passwordInput = screen.getByPlaceholderText('Password');
    
    expect(emailInput).toHaveAttribute('type', 'email');
    expect(passwordInput).toHaveAttribute('type', 'password');
  });

  it('handles non-Error exceptions gracefully', async () => {
    const user = userEvent.setup();
    vi.spyOn(api, 'login').mockRejectedValue('String error');

    render(<LoginForm />);
    
    const emailInput = screen.getByPlaceholderText('Email address');
    const passwordInput = screen.getByPlaceholderText('Password');
    const submitButton = screen.getByRole('button', { name: /sign in/i });
    
    await user.type(emailInput, 'test@jbs.com');
    await user.type(passwordInput, 'password');
    await user.click(submitButton);
    
    await waitFor(() => {
      expect(screen.getByText('Login failed')).toBeInTheDocument();
    });
  });

  it('clears error on new submission', async () => {
    const user = userEvent.setup();
    const mockLogin = vi.spyOn(api, 'login')
      .mockRejectedValueOnce(new Error('Wrong password'))
      .mockResolvedValueOnce({ token: 'token', user: {} as any });

    render(<LoginForm />);
    
    const emailInput = screen.getByPlaceholderText('Email address');
    const passwordInput = screen.getByPlaceholderText('Password');
    const submitButton = screen.getByRole('button', { name: /sign in/i });
    
    await user.type(emailInput, 'test@jbs.com');
    await user.type(passwordInput, 'wrong');
    await user.click(submitButton);
    
    await waitFor(() => {
      expect(screen.getByText('Wrong password')).toBeInTheDocument();
    });
    
    await user.clear(passwordInput);
    await user.type(passwordInput, 'correct');
    await user.click(submitButton);
    
    await waitFor(() => {
      expect(screen.queryByText('Wrong password')).not.toBeInTheDocument();
    });
  });

  it('displays demo credentials', () => {
    render(<LoginForm />);
    
    expect(screen.getByText(/emily\.simpson@jbsconstructiongroup\.com/)).toBeInTheDocument();
  });
});
