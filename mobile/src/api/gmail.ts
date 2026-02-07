import { apiClient } from './client';
import type {
  GmailStatus,
  EmailSource,
  EmailSourceType,
  CreateEmailSourceRequest,
  UpdateEmailSourceRequest,
  TopContact,
  AddCustomSourceRequest,
} from '../types/gmail';

// Re-export types for convenience
export type {
  GmailStatus,
  EmailSource,
  EmailSourceType,
  CreateEmailSourceRequest,
  UpdateEmailSourceRequest,
  TopContact,
  AddCustomSourceRequest,
};

export async function getGmailStatus(): Promise<GmailStatus> {
  return apiClient.get<GmailStatus>('/api/gmail/status');
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

export async function updateEmailSource(id: number, data: UpdateEmailSourceRequest): Promise<EmailSource> {
  return apiClient.put<EmailSource>(`/api/gmail/sources/${id}`, data);
}

export async function deleteEmailSource(id: number): Promise<void> {
  return apiClient.delete<void>(`/api/gmail/sources/${id}`);
}

// Top Contacts API - cached contacts for fast discovery
export async function getTopContacts(): Promise<TopContact[]> {
  const response = await apiClient.get<{ contacts: TopContact[] }>('/api/gmail/top-contacts');
  return response.contacts || [];
}

// Add custom email or domain source
export async function addCustomSource(data: AddCustomSourceRequest): Promise<EmailSource> {
  return apiClient.post<EmailSource>('/api/gmail/sources/custom', data);
}

// Search all cached contacts by name or email
export async function searchGmailContacts(query: string): Promise<TopContact[]> {
  const response = await apiClient.get<{ contacts: TopContact[] }>(
    '/api/gmail/contacts/search',
    { params: { query } }
  );
  return response.contacts || [];
}
