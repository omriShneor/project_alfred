import { apiClient } from './client';
import type { AppStatus, CompleteOnboardingRequest } from '../types/app';

export function getAppStatus(): Promise<AppStatus> {
  return apiClient.get<AppStatus>('/api/app/status');
}

export function completeOnboarding(data: CompleteOnboardingRequest): Promise<AppStatus> {
  return apiClient.post<AppStatus>('/api/onboarding/complete', data);
}
