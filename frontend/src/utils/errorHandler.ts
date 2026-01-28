// Global error handler for consistent error handling across the app
import { toast } from './toast';

export interface ApiError {
  message: string;
  status?: number;
  code?: string;
}

export class ErrorHandler {
  /**
   * Handle API errors with user-friendly messages
   */
  static async handleApiError(error: unknown, context?: string): Promise<void> {
    console.error(`[Error Handler${context ? ` - ${context}` : ''}]:`, error);

    let message = 'An unexpected error occurred';
    
    if (error instanceof Response) {
      // Handle fetch Response errors
      try {
        const data = await error.json();
        message = data.error || data.message || `Server error (${error.status})`;
      } catch {
        message = `Server error (${error.status})`;
      }
    } else if (error instanceof Error) {
      message = error.message;
    } else if (typeof error === 'string') {
      message = error;
    }

    // Show user-friendly error
    toast.error(message);
  }

  /**
   * Handle errors and show custom message
   */
  static handle(error: unknown, customMessage?: string): void {
    console.error('[Error Handler]:', error);
    
    if (customMessage) {
      toast.error(customMessage);
    } else {
      const message = error instanceof Error ? error.message : 'An unexpected error occurred';
      toast.error(message);
    }
  }

  /**
   * Handle network errors
   */
  static handleNetworkError(error: unknown): void {
    console.error('[Network Error]:', error);
    
    if (error instanceof TypeError && error.message.includes('fetch')) {
      toast.error('Network error. Please check your connection and try again.');
    } else {
      this.handle(error, 'Failed to connect to server');
    }
  }

  /**
   * Handle authentication errors
   */
  static handleAuthError(error: unknown): void {
    console.error('[Auth Error]:', error);
    toast.error('Session expired. Please log in again.');
    
    // Clear auth and redirect
    localStorage.removeItem('jbs_token');
    localStorage.removeItem('jbs_user');
    
    setTimeout(() => {
      window.location.href = '/';
    }, 1500);
  }

  /**
   * Handle validation errors
   */
  static handleValidationError(field: string, message: string): void {
    toast.warning(`${field}: ${message}`);
  }

  /**
   * Global error handler for uncaught errors
   */
  static initGlobalHandler(): void {
    // Handle uncaught errors
    window.addEventListener('error', (event) => {
      console.error('[Global Error]:', event.error);
      toast.error('An unexpected error occurred. Please refresh the page.');
      event.preventDefault();
    });

    // Handle unhandled promise rejections
    window.addEventListener('unhandledrejection', (event) => {
      console.error('[Unhandled Rejection]:', event.reason);
      this.handle(event.reason, 'An error occurred. Please try again.');
      event.preventDefault();
    });
  }

  /**
   * Check if error is an authentication error
   */
  static isAuthError(error: unknown): boolean {
    if (error instanceof Response) {
      return error.status === 401 || error.status === 403;
    }
    return false;
  }

  /**
   * Safely execute async function with error handling
   */
  static async tryAsync<T>(
    fn: () => Promise<T>,
    errorMessage?: string
  ): Promise<T | null> {
    try {
      return await fn();
    } catch (error) {
      if (this.isAuthError(error)) {
        this.handleAuthError(error);
      } else {
        this.handle(error, errorMessage);
      }
      return null;
    }
  }
}

// Initialize global error handler
if (typeof window !== 'undefined') {
  ErrorHandler.initGlobalHandler();
}
