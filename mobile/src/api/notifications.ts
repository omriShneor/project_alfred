import { apiClient } from './client';

export interface NotificationPreferences {
  email_enabled: boolean;
  email_address: string;
  push_enabled: boolean;
  push_token?: string;
  sms_enabled: boolean;
  webhook_enabled: boolean;
}

export interface NotificationPrefsResponse {
  preferences: NotificationPreferences;
  available: {
    email: boolean;
    push: boolean;
    sms: boolean;
    webhook: boolean;
  };
}

export async function getNotificationPrefs(): Promise<NotificationPrefsResponse> {
  const response = await apiClient.get<NotificationPrefsResponse>('/api/notifications/preferences');
  return response.data;
}

export async function updateEmailPrefs(enabled: boolean, address: string): Promise<NotificationPreferences> {
  const response = await apiClient.put<NotificationPreferences>('/api/notifications/email', {
    enabled,
    address,
  });
  return response.data;
}

export async function registerPushToken(token: string): Promise<{ status: string }> {
  const response = await apiClient.post<{ status: string }>('/api/notifications/push/register', {
    token,
  });
  return response.data;
}

export async function updatePushPrefs(enabled: boolean): Promise<NotificationPreferences> {
  const response = await apiClient.put<NotificationPreferences>('/api/notifications/push', {
    enabled,
  });
  return response.data;
}
