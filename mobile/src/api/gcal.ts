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

export async function disconnectGCal(): Promise<void> {
  await apiClient.post('/api/gcal/disconnect');
}

// Global Google Calendar Settings

export interface GCalSettings {
  id: number;
  sync_enabled: boolean;
  selected_calendar_id: string;
  selected_calendar_name: string;
  created_at: string;
  updated_at: string;
}

export interface UpdateGCalSettingsRequest {
  sync_enabled: boolean;
  selected_calendar_id: string;
  selected_calendar_name: string;
}

export async function getGCalSettings(): Promise<GCalSettings> {
  return apiClient.get<GCalSettings>('/api/gcal/settings');
}

export async function updateGCalSettings(data: UpdateGCalSettingsRequest): Promise<GCalSettings> {
  return apiClient.put<GCalSettings>('/api/gcal/settings', data);
}

// Note: listCalendars is exported from events.ts to avoid duplication
