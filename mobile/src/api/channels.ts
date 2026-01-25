import { apiClient } from './client';
import type {
  Channel,
  DiscoverableChannel,
  CreateChannelRequest,
  UpdateChannelRequest,
} from '../types/channel';

export async function listChannels(type?: string): Promise<Channel[]> {
  const params = type ? { type } : undefined;
  const response = await apiClient.get<Channel[]>('/api/channel', { params });
  return response.data;
}

export async function createChannel(data: CreateChannelRequest): Promise<Channel> {
  const response = await apiClient.post<Channel>('/api/channel', data);
  return response.data;
}

export async function updateChannel(
  id: number,
  data: UpdateChannelRequest
): Promise<Channel> {
  const response = await apiClient.put<Channel>(`/api/channel/${id}`, data);
  return response.data;
}

export async function deleteChannel(id: number): Promise<void> {
  await apiClient.delete(`/api/channel/${id}`);
}

export async function discoverChannels(): Promise<DiscoverableChannel[]> {
  const response = await apiClient.get<DiscoverableChannel[]>(
    '/api/discovery/channels'
  );
  return response.data;
}
