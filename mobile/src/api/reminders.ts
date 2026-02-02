import { apiClient } from './client';
import type {
  Reminder,
  ReminderWithMessage,
  UpdateReminderRequest,
} from '../types/reminder';

export interface ListRemindersParams {
  status?: string;
  channel_id?: number;
}

export async function listReminders(params?: ListRemindersParams): Promise<Reminder[]> {
  return apiClient.get<Reminder[]>('/api/reminders', { params: params as Record<string, string | number | undefined> });
}

export async function getReminder(id: number): Promise<ReminderWithMessage> {
  return apiClient.get<ReminderWithMessage>(`/api/reminders/${id}`);
}

export async function updateReminder(
  id: number,
  data: UpdateReminderRequest
): Promise<Reminder> {
  return apiClient.put<Reminder>(`/api/reminders/${id}`, data);
}

export async function confirmReminder(id: number): Promise<Reminder> {
  return apiClient.post<Reminder>(`/api/reminders/${id}/confirm`);
}

export async function rejectReminder(id: number): Promise<Reminder> {
  return apiClient.post<Reminder>(`/api/reminders/${id}/reject`);
}

export async function completeReminder(id: number): Promise<Reminder> {
  return apiClient.post<Reminder>(`/api/reminders/${id}/complete`);
}

export async function dismissReminder(id: number): Promise<Reminder> {
  return apiClient.post<Reminder>(`/api/reminders/${id}/dismiss`);
}
