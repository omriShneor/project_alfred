import { apiClient } from './client';
import type { TodayEvent } from '../types/calendar';

export async function getTodayEvents(calendarId?: string): Promise<TodayEvent[]> {
  const params = calendarId ? { calendar_id: calendarId } : {};
  const response = await apiClient.get<TodayEvent[]>('/api/gcal/events/today', { params });
  return response.data;
}
