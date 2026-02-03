import { API_BASE_URL } from '../config/api';
import { getSessionToken, clearAllAuthData } from '../auth/storage';

const TIMEOUT = 30000;

// Event emitter for auth state changes
type AuthEventListener = () => void;
const authListeners: AuthEventListener[] = [];

export function onAuthError(listener: AuthEventListener) {
  authListeners.push(listener);
  return () => {
    const index = authListeners.indexOf(listener);
    if (index > -1) authListeners.splice(index, 1);
  };
}

function notifyAuthError() {
  authListeners.forEach((listener) => listener());
}

interface RequestOptions {
  params?: Record<string, string | number | undefined>;
  body?: unknown;
  skipAuth?: boolean; // For public endpoints like /health
}

async function request<T>(
  method: string,
  path: string,
  options: RequestOptions = {}
): Promise<T> {
  const { params, body, skipAuth = false } = options;

  // Build URL with query params
  let url = `${API_BASE_URL}${path}`;
  if (params) {
    const searchParams = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined) {
        searchParams.append(key, String(value));
      }
    });
    const queryString = searchParams.toString();
    if (queryString) {
      url += `?${queryString}`;
    }
  }

  if (__DEV__) {
    console.log(`[API] ${method} ${path}`);
  }

  // Build headers
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  // Add auth header if we have a token and auth is not skipped
  if (!skipAuth) {
    const token = await getSessionToken();
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }
  }

  // Create abort controller for timeout
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), TIMEOUT);

  try {
    const response = await fetch(url, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
      signal: controller.signal,
    });

    clearTimeout(timeoutId);

    // Handle 401 Unauthorized - session expired
    if (response.status === 401) {
      if (__DEV__) {
        console.log('[API] Session expired, clearing auth data');
      }
      await clearAllAuthData();
      notifyAuthError();
      throw new AuthError('Session expired');
    }

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      if (__DEV__) {
        console.error('[API Error]', errorData);
      }
      throw new Error(errorData.error || `HTTP ${response.status}`);
    }

    // Handle empty responses (204 No Content)
    if (response.status === 204) {
      return undefined as T;
    }

    return response.json();
  } catch (error) {
    clearTimeout(timeoutId);
    if (error instanceof Error && error.name === 'AbortError') {
      throw new Error('Request timeout');
    }
    if (__DEV__) {
      console.error('[API Error]', error);
    }
    throw error;
  }
}

export class AuthError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'AuthError';
  }
}

export const apiClient = {
  get<T>(
    path: string,
    options?: { params?: Record<string, string | number | undefined>; skipAuth?: boolean }
  ): Promise<T> {
    return request<T>('GET', path, options);
  },

  post<T>(path: string, body?: unknown, options?: { skipAuth?: boolean }): Promise<T> {
    return request<T>('POST', path, { body, ...options });
  },

  put<T>(path: string, body?: unknown, options?: { skipAuth?: boolean }): Promise<T> {
    return request<T>('PUT', path, { body, ...options });
  },

  delete<T>(path: string, options?: { skipAuth?: boolean }): Promise<T> {
    return request<T>('DELETE', path, options);
  },
};
