import Constants from 'expo-constants';

// API Configuration
// Reads from EXPO_PUBLIC_API_BASE_URL environment variable
// Falls back to localhost for local development

export const API_BASE_URL =
  Constants.expoConfig?.extra?.apiBaseUrl || 'http://localhost:8080';

// Web settings URL (for opening in browser when there are connection issues)
export const WEB_SETTINGS_URL = `${API_BASE_URL}/settings`;
