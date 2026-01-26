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
  const response = await apiClient.get<GCalStatus>('/api/gcal/status');
  return response.data;
}

export async function getOAuthURL(redirectUri: string): Promise<GCalConnectResponse> {
  const response = await apiClient.post<GCalConnectResponse>('/api/gcal/connect', {
    redirect_uri: redirectUri,
  });
  return response.data;
}

export async function exchangeOAuthCode(code: string, redirectUri: string): Promise<void> {
  await apiClient.post('/api/gcal/callback', {
    code,
    redirect_uri: redirectUri,
  });
}

// Note: listCalendars is exported from events.ts to avoid duplication
