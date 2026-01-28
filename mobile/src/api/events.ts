import { apiClient } from './client';
import type {
  CalendarEvent,
  EventWithMessage,
  UpdateEventRequest,
  MessageHistory,
  Calendar,
} from '../types/event';

export interface ListEventsParams {
  status?: string;
  channel_id?: number;
}

export async function listEvents(params?: ListEventsParams): Promise<CalendarEvent[]> {
  return apiClient.get<CalendarEvent[]>('/api/events', { params: params as Record<string, string | number | undefined> });
}

export async function getEvent(id: number): Promise<EventWithMessage> {
  return apiClient.get<EventWithMessage>(`/api/events/${id}`);
}

export async function updateEvent(
  id: number,
  data: UpdateEventRequest
): Promise<CalendarEvent> {
  return apiClient.put<CalendarEvent>(`/api/events/${id}`, data);
}

export async function confirmEvent(id: number): Promise<CalendarEvent> {
  return apiClient.post<CalendarEvent>(`/api/events/${id}/confirm`);
}

export async function rejectEvent(id: number): Promise<CalendarEvent> {
  return apiClient.post<CalendarEvent>(`/api/events/${id}/reject`);
}

export async function getChannelHistory(
  channelId: number
): Promise<MessageHistory[]> {
  return apiClient.get<MessageHistory[]>(
    `/api/events/channel/${channelId}/history`
  );
}

export async function listCalendars(): Promise<Calendar[]> {
  return apiClient.get<Calendar[]>('/api/gcal/calendars');
}
