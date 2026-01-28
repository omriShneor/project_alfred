import { API_BASE_URL } from '../config/api';

const TIMEOUT = 30000;

interface RequestOptions {
  params?: Record<string, string | number | undefined>;
  body?: unknown;
}

async function request<T>(
  method: string,
  path: string,
  options: RequestOptions = {}
): Promise<T> {
  const { params, body } = options;

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

  // Create abort controller for timeout
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), TIMEOUT);

  try {
    const response = await fetch(url, {
      method,
      headers: {
        'Content-Type': 'application/json',
      },
      body: body ? JSON.stringify(body) : undefined,
      signal: controller.signal,
    });

    clearTimeout(timeoutId);

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

export const apiClient = {
  get<T>(path: string, options?: { params?: Record<string, string | number | undefined> }): Promise<T> {
    return request<T>('GET', path, options);
  },

  post<T>(path: string, body?: unknown): Promise<T> {
    return request<T>('POST', path, { body });
  },

  put<T>(path: string, body?: unknown): Promise<T> {
    return request<T>('PUT', path, { body });
  },

  delete<T>(path: string): Promise<T> {
    return request<T>('DELETE', path);
  },
};
