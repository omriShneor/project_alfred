import { Platform } from 'react-native';

const SESSION_KEY = 'alfred_session_token';
const USER_KEY = 'alfred_user';

// Web fallback using localStorage (less secure but works for development)
const webStorage = {
  async getItem(key: string): Promise<string | null> {
    if (typeof window !== 'undefined' && window.localStorage) {
      return window.localStorage.getItem(key);
    }
    return null;
  },
  async setItem(key: string, value: string): Promise<void> {
    if (typeof window !== 'undefined' && window.localStorage) {
      window.localStorage.setItem(key, value);
    }
  },
  async deleteItem(key: string): Promise<void> {
    if (typeof window !== 'undefined' && window.localStorage) {
      window.localStorage.removeItem(key);
    }
  },
};

// Storage interface
interface StorageInterface {
  getItem: (key: string) => Promise<string | null>;
  setItem: (key: string, value: string) => Promise<void>;
  deleteItem: (key: string) => Promise<void>;
}

// Cached secure store instance
let secureStoreInstance: StorageInterface | null = null;

// Use expo-secure-store on native platforms, localStorage on web
async function getSecureStore(): Promise<StorageInterface> {
  if (secureStoreInstance) {
    return secureStoreInstance;
  }

  if (Platform.OS === 'web') {
    secureStoreInstance = webStorage;
    return secureStoreInstance;
  }

  try {
    // Dynamic import for native platforms only
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const SecureStore = require('expo-secure-store');
    secureStoreInstance = {
      getItem: SecureStore.getItemAsync,
      setItem: SecureStore.setItemAsync,
      deleteItem: SecureStore.deleteItemAsync,
    };
    return secureStoreInstance;
  } catch {
    // Fallback to web storage if secure store fails
    console.warn('expo-secure-store not available, using localStorage fallback');
    secureStoreInstance = webStorage;
    return secureStoreInstance;
  }
}

export interface StoredUser {
  id: number;
  email: string;
  name: string;
  avatarUrl?: string;
}

export async function getSessionToken(): Promise<string | null> {
  const store = await getSecureStore();
  return store.getItem(SESSION_KEY);
}

export async function setSessionToken(token: string): Promise<void> {
  const store = await getSecureStore();
  await store.setItem(SESSION_KEY, token);
}

export async function clearSessionToken(): Promise<void> {
  const store = await getSecureStore();
  await store.deleteItem(SESSION_KEY);
}

export async function getStoredUser(): Promise<StoredUser | null> {
  const store = await getSecureStore();
  const userJson = await store.getItem(USER_KEY);
  if (userJson) {
    try {
      return JSON.parse(userJson);
    } catch {
      return null;
    }
  }
  return null;
}

export async function setStoredUser(user: StoredUser): Promise<void> {
  const store = await getSecureStore();
  await store.setItem(USER_KEY, JSON.stringify(user));
}

export async function clearStoredUser(): Promise<void> {
  const store = await getSecureStore();
  await store.deleteItem(USER_KEY);
}

export async function clearAllAuthData(): Promise<void> {
  await Promise.all([clearSessionToken(), clearStoredUser()]);
}
