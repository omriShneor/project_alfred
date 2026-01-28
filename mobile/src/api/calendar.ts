import { apiClient } from './client';
import type { TodayEvent } from '../types/calendar';

// Get merged today's events from Alfred Calendar + external calendars (Google, etc.)
// This is the primary endpoint for Today's Schedule
export async function getTodayEvents(calendarId?: string): Promise<TodayEvent[]> {
  const params = calendarId ? { calendar_id: calendarId } : {};
  return apiClient.get<TodayEvent[]>('/api/events/today', { params });
}

// Get today's events from Google Calendar only (legacy endpoint)
export async function getGoogleTodayEvents(calendarId?: string): Promise<TodayEvent[]> {
  const params = calendarId ? { calendar_id: calendarId } : {};
  return apiClient.get<TodayEvent[]>('/api/gcal/events/today', { params });
}
