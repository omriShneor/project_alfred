import { apiClient } from './client';

export interface WhatsAppStatus {
  connected: boolean;
  message: string;
}

export interface PairingCodeResponse {
  code: string;
  message: string;
}

export async function getWhatsAppStatus(): Promise<WhatsAppStatus> {
  const response = await apiClient.get<WhatsAppStatus>('/api/whatsapp/status');
  return response.data;
}

export async function generatePairingCode(phoneNumber: string): Promise<PairingCodeResponse> {
  const response = await apiClient.post<PairingCodeResponse>('/api/whatsapp/pair', {
    phone_number: phoneNumber,
  });
  return response.data;
}

export async function disconnectWhatsApp(): Promise<void> {
  await apiClient.post('/api/whatsapp/disconnect');
}

export async function reconnectWhatsApp(): Promise<void> {
  await apiClient.post('/api/whatsapp/reconnect');
}
