import { apiClient } from './client';

export interface HealthStatus {
  status: string;
  whatsapp: 'connected' | 'disconnected';
  gcal: 'connected' | 'disconnected';
}

export async function getHealth(): Promise<HealthStatus> {
  // Health check is a public endpoint, skip auth
  return apiClient.get<HealthStatus>('/health', { skipAuth: true });
}
