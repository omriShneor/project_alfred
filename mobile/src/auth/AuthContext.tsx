import React, { createContext, useContext, useEffect, useState, useCallback } from 'react';
import {
  getSessionToken,
  setSessionToken,
  clearAllAuthData,
  getStoredUser,
  setStoredUser,
  clearStoredUser,
  StoredUser,
} from './storage';
import { API_BASE_URL } from '../config/api';

export interface User {
  id: number;
  email: string;
  name: string;
  avatarUrl?: string;
  timezone?: string;
}

interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (code: string, redirectUri: string) => Promise<void>;
  logout: () => Promise<void>;
  getToken: () => Promise<string | null>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

interface AuthProviderProps {
  children: React.ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Load stored auth state on mount
  useEffect(() => {
    async function loadAuthState() {
      try {
        const [token, storedUser] = await Promise.all([
          getSessionToken(),
          getStoredUser(),
        ]);

        if (token && storedUser) {
          // Validate token is still valid by calling /api/auth/me
          try {
            const response = await fetch(`${API_BASE_URL}/api/auth/me`, {
              headers: {
                Authorization: `Bearer ${token}`,
              },
            });

            if (response.ok) {
              const userData = await response.json();
              setUser({
                id: userData.id,
                email: userData.email,
                name: userData.name,
                avatarUrl: userData.avatar_url,
                timezone: userData.timezone,
              });
            } else {
              // Token invalid, clear auth data
              await clearAllAuthData();
            }
          } catch {
            // Network error, use stored user data
            setUser({
              id: storedUser.id,
              email: storedUser.email,
              name: storedUser.name,
              avatarUrl: storedUser.avatarUrl,
              timezone: storedUser.timezone,
            });
          }
        }
      } finally {
        setIsLoading(false);
      }
    }

    loadAuthState();
  }, []);

  const login = useCallback(async (code: string, redirectUri: string) => {
    setIsLoading(true);
    try {
      // Exchange the OAuth code for a session token
      const response = await fetch(`${API_BASE_URL}/api/auth/google/callback`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ code, redirect_uri: redirectUri }),
      });

      if (!response.ok) {
        const error = await response.json().catch(() => ({}));
        throw new Error(error.error || 'Login failed');
      }

      const data = await response.json();

      // Store the session token and user
      await setSessionToken(data.session_token);

      const newUser: User = {
        id: data.user.id,
        email: data.user.email,
        name: data.user.name,
        avatarUrl: data.user.avatar_url,
        timezone: data.user.timezone,
      };

      await setStoredUser({
        id: newUser.id,
        email: newUser.email,
        name: newUser.name,
        avatarUrl: newUser.avatarUrl,
        timezone: newUser.timezone,
      });

      setUser(newUser);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const logout = useCallback(async () => {
    setIsLoading(true);
    try {
      const token = await getSessionToken();
      if (token) {
        // Notify server of logout
        try {
          await fetch(`${API_BASE_URL}/api/auth/google/logout`, {
            method: 'POST',
            headers: {
              Authorization: `Bearer ${token}`,
            },
          });
        } catch {
          // Ignore network errors during logout
        }
      }

      await clearAllAuthData();
      setUser(null);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const getToken = useCallback(async () => {
    return getSessionToken();
  }, []);

  const value: AuthContextType = {
    user,
    isAuthenticated: !!user,
    isLoading,
    login,
    logout,
    getToken,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextType {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
