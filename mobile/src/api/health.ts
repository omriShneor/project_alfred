import { apiClient } from './client';

export interface HealthStatus {
  status: string;
  whatsapp: 'connected' | 'disconnected';
  gcal: 'connected' | 'disconnected';
}

export async function getHealth(): Promise<HealthStatus> {
  const response = await apiClient.get<HealthStatus>('/health');
  return response.data;
}
