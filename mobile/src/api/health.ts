import { apiClient } from './client';

export interface HealthStatus {
  status: string;
  whatsapp: 'connected' | 'disconnected';
  gcal: 'connected' | 'disconnected';
}

export async function getHealth(): Promise<HealthStatus> {
  return apiClient.get<HealthStatus>('/health');
}
