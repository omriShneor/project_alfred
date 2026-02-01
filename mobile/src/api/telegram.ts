import { apiClient } from './client';
import type { Channel, DiscoverableChannel, SourceTopContact } from '../types/channel';

// Telegram status response
export interface TelegramStatus {
  connected: boolean;
  message?: string;
}

// Get Telegram connection status
export async function getTelegramStatus(): Promise<TelegramStatus> {
  return apiClient.get<TelegramStatus>('/api/telegram/status');
}

// Send verification code to phone number
export async function sendTelegramCode(phoneNumber: string): Promise<{ message: string }> {
  return apiClient.post<{ message: string }>('/api/telegram/send-code', { phone_number: phoneNumber });
}

// Verify code and complete authentication
export async function verifyTelegramCode(code: string): Promise<TelegramStatus> {
  return apiClient.post<TelegramStatus>('/api/telegram/verify-code', { code });
}

// Disconnect Telegram
export async function disconnectTelegram(): Promise<void> {
  await apiClient.post('/api/telegram/disconnect');
}

// Reconnect Telegram
export async function reconnectTelegram(): Promise<TelegramStatus> {
  return apiClient.post<TelegramStatus>('/api/telegram/reconnect');
}

// Discover available Telegram channels
export async function discoverTelegramChannels(): Promise<DiscoverableChannel[]> {
  return apiClient.get<DiscoverableChannel[]>('/api/telegram/discovery/channels');
}

// List tracked Telegram channels
export async function listTelegramChannels(): Promise<Channel[]> {
  return apiClient.get<Channel[]>('/api/telegram/channel');
}

// Create a tracked Telegram channel
export interface CreateTelegramChannelRequest {
  type: 'contact' | 'group' | 'channel';
  identifier: string;
  name: string;
}

export async function createTelegramChannel(data: CreateTelegramChannelRequest): Promise<Channel> {
  return apiClient.post<Channel>('/api/telegram/channel', data);
}

// Update a tracked Telegram channel
export interface UpdateTelegramChannelRequest {
  name: string;
  enabled: boolean;
}

export async function updateTelegramChannel(id: number, data: UpdateTelegramChannelRequest): Promise<Channel> {
  return apiClient.put<Channel>(`/api/telegram/channel/${id}`, data);
}

// Delete a tracked Telegram channel
export async function deleteTelegramChannel(id: number): Promise<void> {
  await apiClient.delete(`/api/telegram/channel/${id}`);
}

// Get top Telegram contacts based on message frequency
export async function getTelegramTopContacts(): Promise<SourceTopContact[]> {
  const response = await apiClient.get<{ contacts: SourceTopContact[] }>('/api/telegram/top-contacts');
  return response.contacts || [];
}

// Add a custom Telegram source by username
export async function addTelegramCustomSource(username: string): Promise<Channel> {
  return apiClient.post<Channel>('/api/telegram/sources/custom', {
    username,
  });
}
