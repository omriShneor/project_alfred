import { apiClient } from './client';
import type {
  Channel,
  CreateChannelRequest,
  UpdateChannelRequest,
  SourceTopContact,
} from '../types/channel';

export async function listChannels(type?: string): Promise<Channel[]> {
  const params = type ? { type } : undefined;
  return apiClient.get<Channel[]>('/api/whatsapp/channel', { params });
}

export async function createChannel(data: CreateChannelRequest): Promise<Channel> {
  return apiClient.post<Channel>('/api/whatsapp/channel', data);
}

export async function updateChannel(
  id: number,
  data: UpdateChannelRequest
): Promise<Channel> {
  return apiClient.put<Channel>(`/api/whatsapp/channel/${id}`, data);
}

export async function deleteChannel(id: number): Promise<void> {
  await apiClient.delete(`/api/whatsapp/channel/${id}`);
}

export async function getWhatsAppTopContacts(): Promise<SourceTopContact[]> {
  const response = await apiClient.get<{ contacts: SourceTopContact[] }>('/api/whatsapp/top-contacts');
  return response.contacts || [];
}

export async function addWhatsAppCustomSource(contactName: string): Promise<Channel> {
  return apiClient.post<Channel>('/api/whatsapp/sources/custom', {
    name: contactName,
  });
}

export async function searchWhatsAppContacts(query: string): Promise<SourceTopContact[]> {
  const response = await apiClient.get<{ contacts: SourceTopContact[] }>(
    '/api/whatsapp/contacts/search',
    { params: { query } }
  );
  return response.contacts || [];
}
