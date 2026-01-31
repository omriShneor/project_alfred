import { apiClient } from './client';
import type {
  Channel,
  DiscoverableChannel,
  CreateChannelRequest,
  UpdateChannelRequest,
  SourceTopContact,
} from '../types/channel';

export async function listChannels(type?: string): Promise<Channel[]> {
  const params = type ? { type } : undefined;
  return apiClient.get<Channel[]>('/api/channel', { params });
}

export async function createChannel(data: CreateChannelRequest): Promise<Channel> {
  return apiClient.post<Channel>('/api/channel', data);
}

export async function updateChannel(
  id: number,
  data: UpdateChannelRequest
): Promise<Channel> {
  return apiClient.put<Channel>(`/api/channel/${id}`, data);
}

export async function deleteChannel(id: number): Promise<void> {
  await apiClient.delete(`/api/channel/${id}`);
}

export async function discoverChannels(): Promise<DiscoverableChannel[]> {
  return apiClient.get<DiscoverableChannel[]>(
    '/api/discovery/channels'
  );
}

// WhatsApp Top Contacts API
export async function getWhatsAppTopContacts(): Promise<SourceTopContact[]> {
  const response = await apiClient.get<{ contacts: SourceTopContact[] }>('/api/whatsapp/top-contacts');
  return response.contacts || [];
}

export async function addWhatsAppCustomSource(phoneNumber: string, calendarId: string): Promise<Channel> {
  return apiClient.post<Channel>('/api/whatsapp/sources/custom', {
    phone_number: phoneNumber,
    calendar_id: calendarId,
  });
}
