import { apiClient } from './client';

export interface OnboardingStatus {
  whatsapp: {
    status: string;
    qr?: string;
    error?: string;
  };
  gcal: {
    status: string;
    configured: boolean;
    error?: string;
  };
  complete: boolean;
}

export async function getOnboardingStatus(): Promise<OnboardingStatus> {
  const response = await apiClient.get<OnboardingStatus>('/api/onboarding/status');
  return response.data;
}
