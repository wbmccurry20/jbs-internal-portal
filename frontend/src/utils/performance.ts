// Utility functions for performance optimization

/**
 * Debounce function to limit API calls from search inputs
 * @param func - Function to debounce
 * @param wait - Milliseconds to wait
 * @returns Debounced function
 */
export function debounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeout: ReturnType<typeof setTimeout> | null = null;
  
  return function executedFunction(...args: Parameters<T>) {
    const later = () => {
      timeout = null;
      func(...args);
    };
    
    if (timeout) {
      clearTimeout(timeout);
    }
    timeout = setTimeout(later, wait);
  };
}

/**
 * Throttle function to limit function calls
 * @param func - Function to throttle  
 * @param limit - Minimum milliseconds between calls
 * @returns Throttled function
 */
export function throttle<T extends (...args: any[]) => any>(
  func: T,
  limit: number
): (...args: Parameters<T>) => void {
  let inThrottle: boolean;
  
  return function executedFunction(...args: Parameters<T>) {
    if (!inThrottle) {
      func(...args);
      inThrottle = true;
      setTimeout(() => (inThrottle = false), limit);
    }
  };
}

/**
 * Simple in-memory cache with expiration
 */
export class SimpleCache<T> {
  private cache: Map<string, { data: T; expires: number }> = new Map();
  
  constructor(private ttl: number = 5 * 60 * 1000) {} // 5 min default
  
  get(key: string): T | null {
    const entry = this.cache.get(key);
    if (!entry) return null;
    
    if (Date.now() > entry.expires) {
      this.cache.delete(key);
      return null;
    }
    
    return entry.data;
  }
  
  set(key: string, data: T): void {
    this.cache.set(key, {
      data,
      expires: Date.now() + this.ttl,
    });
  }
  
  clear(): void {
    this.cache.clear();
  }
  
  clearPattern(pattern: string): void {
    for (const key of this.cache.keys()) {
      if (key.includes(pattern)) {
        this.cache.delete(key);
      }
    }
  }
}

/**
 * Fetch with caching
 */
export async function cachedFetch<T>(
  url: string,
  cache: SimpleCache<T>,
  options?: RequestInit
): Promise<T> {
  // Check cache first
  const cached = cache.get(url);
  if (cached) {
    console.log(`[Cache HIT] ${url}`);
    return cached;
  }
  
  console.log(`[Cache MISS] ${url}`);
  const response = await fetch(url, options);
  
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }
  
  const data = await response.json() as T;
  cache.set(url, data);
  
  return data;
}
