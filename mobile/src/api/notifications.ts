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
  return apiClient.get<NotificationPrefsResponse>('/api/notifications/preferences');
}

export async function updateEmailPrefs(enabled: boolean, address: string): Promise<NotificationPreferences> {
  return apiClient.put<NotificationPreferences>('/api/notifications/email', {
    enabled,
    address,
  });
}

export async function registerPushToken(token: string): Promise<{ status: string }> {
  return apiClient.post<{ status: string }>('/api/notifications/push/register', {
    token,
  });
}

export async function updatePushPrefs(enabled: boolean): Promise<NotificationPreferences> {
  return apiClient.put<NotificationPreferences>('/api/notifications/push', {
    enabled,
  });
}
