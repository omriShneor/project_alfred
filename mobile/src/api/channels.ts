import { apiClient } from './client';
import type {
  Channel,
  DiscoverableChannel,
  CreateChannelRequest,
  UpdateChannelRequest,
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
