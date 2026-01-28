import { apiClient } from './client';
import type {
  GmailStatus,
  GmailSettings,
  EmailSource,
  EmailSourceType,
  CreateEmailSourceRequest,
  UpdateEmailSourceRequest,
  DiscoveredCategory,
  DiscoveredSender,
  DiscoveredDomain,
} from '../types/gmail';

// Re-export types for convenience
export type {
  GmailStatus,
  GmailSettings,
  EmailSource,
  EmailSourceType,
  CreateEmailSourceRequest,
  UpdateEmailSourceRequest,
  DiscoveredCategory,
  DiscoveredSender,
  DiscoveredDomain,
};

export async function getGmailStatus(): Promise<GmailStatus> {
  return apiClient.get<GmailStatus>('/api/gmail/status');
}

export async function getGmailSettings(): Promise<GmailSettings> {
  return apiClient.get<GmailSettings>('/api/gmail/settings');
}

export async function updateGmailSettings(settings: Partial<GmailSettings>): Promise<GmailSettings> {
  return apiClient.put<GmailSettings>('/api/gmail/settings', settings);
}

export async function discoverCategories(): Promise<DiscoveredCategory[]> {
  const response = await apiClient.get<{ categories: DiscoveredCategory[] }>('/api/gmail/discover/categories');
  return response.categories || [];
}

export async function discoverSenders(limit?: number): Promise<DiscoveredSender[]> {
  const response = await apiClient.get<{ senders: DiscoveredSender[] }>('/api/gmail/discover/senders', {
    params: limit ? { limit } : undefined,
  });
  return response.senders || [];
}

export async function discoverDomains(limit?: number): Promise<DiscoveredDomain[]> {
  const response = await apiClient.get<{ domains: DiscoveredDomain[] }>('/api/gmail/discover/domains', {
    params: limit ? { limit } : undefined,
  });
  return response.domains || [];
}

export async function listEmailSources(type?: EmailSourceType): Promise<EmailSource[]> {
  const response = await apiClient.get<{ sources: EmailSource[] }>('/api/gmail/sources', {
    params: type ? { type } : undefined,
  });
  return response.sources || [];
}

export async function createEmailSource(data: CreateEmailSourceRequest): Promise<EmailSource> {
  return apiClient.post<EmailSource>('/api/gmail/sources', data);
}

export async function getEmailSource(id: number): Promise<EmailSource> {
  return apiClient.get<EmailSource>(`/api/gmail/sources/${id}`);
}

export async function updateEmailSource(id: number, data: UpdateEmailSourceRequest): Promise<EmailSource> {
  return apiClient.put<EmailSource>(`/api/gmail/sources/${id}`, data);
}

export async function deleteEmailSource(id: number): Promise<void> {
  return apiClient.delete<void>(`/api/gmail/sources/${id}`);
}
