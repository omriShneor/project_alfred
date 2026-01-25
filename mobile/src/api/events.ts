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
  const response = await apiClient.get<CalendarEvent[]>('/api/events', { params });
  return response.data;
}

export async function getEvent(id: number): Promise<EventWithMessage> {
  const response = await apiClient.get<EventWithMessage>(`/api/events/${id}`);
  return response.data;
}

export async function updateEvent(
  id: number,
  data: UpdateEventRequest
): Promise<CalendarEvent> {
  const response = await apiClient.put<CalendarEvent>(`/api/events/${id}`, data);
  return response.data;
}

export async function confirmEvent(id: number): Promise<CalendarEvent> {
  const response = await apiClient.post<CalendarEvent>(`/api/events/${id}/confirm`);
  return response.data;
}

export async function rejectEvent(id: number): Promise<CalendarEvent> {
  const response = await apiClient.post<CalendarEvent>(`/api/events/${id}/reject`);
  return response.data;
}

export async function getChannelHistory(
  channelId: number
): Promise<MessageHistory[]> {
  const response = await apiClient.get<MessageHistory[]>(
    `/api/events/channel/${channelId}/history`
  );
  return response.data;
}

export async function listCalendars(): Promise<Calendar[]> {
  const response = await apiClient.get<Calendar[]>('/api/gcal/calendars');
  return response.data;
}
