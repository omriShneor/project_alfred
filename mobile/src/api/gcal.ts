import { apiClient } from './client';

export interface GCalStatus {
  connected: boolean;
  message: string;
}

export interface GCalConnectResponse {
  auth_url: string;
  redirect_uri?: string;
  message: string;
}

export interface GCalCalendar {
  id: string;
  summary: string;
  primary: boolean;
}

export async function getGCalStatus(): Promise<GCalStatus> {
  return apiClient.get<GCalStatus>('/api/gcal/status');
}

export async function getOAuthURL(redirectUri?: string): Promise<GCalConnectResponse> {
  // If no redirectUri is provided, backend will use its own HTTPS callback
  return apiClient.post<GCalConnectResponse>('/api/gcal/connect',
    redirectUri ? { redirect_uri: redirectUri } : {}
  );
}

export async function exchangeOAuthCode(code: string, redirectUri?: string): Promise<void> {
  await apiClient.post('/api/gcal/callback',
    redirectUri ? { code, redirect_uri: redirectUri } : { code }
  );
}

// Note: listCalendars is exported from events.ts to avoid duplication
