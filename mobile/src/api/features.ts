import { apiClient } from './client';
import type {
  FeaturesResponse,
  UpdateSmartCalendarRequest,
  SmartCalendarStatusResponse,
} from '../types/features';

// Get all feature settings
export function getFeatures(): Promise<FeaturesResponse> {
  return apiClient.get<FeaturesResponse>('/api/features');
}

// Update Smart Calendar settings
export function updateSmartCalendar(
  data: UpdateSmartCalendarRequest
): Promise<FeaturesResponse> {
  return apiClient.put<FeaturesResponse>('/api/features/smart-calendar', data);
}

// Get detailed Smart Calendar status
export function getSmartCalendarStatus(): Promise<SmartCalendarStatusResponse> {
  return apiClient.get<SmartCalendarStatusResponse>('/api/features/smart-calendar/status');
}
