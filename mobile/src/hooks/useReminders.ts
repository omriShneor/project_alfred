import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  listReminders,
  getReminder,
  updateReminder,
  confirmReminder,
  rejectReminder,
  completeReminder,
  dismissReminder,
  type ListRemindersParams,
} from '../api/reminders';
import type {
  Reminder,
  ReminderWithMessage,
  UpdateReminderRequest,
} from '../types/reminder';

export function useReminders(params?: ListRemindersParams) {
  return useQuery<Reminder[]>({
    queryKey: ['reminders', params],
    queryFn: () => listReminders(params),
  });
}

export function useReminder(id: number) {
  return useQuery<ReminderWithMessage>({
    queryKey: ['reminder', id],
    queryFn: () => getReminder(id),
    enabled: id > 0,
  });
}

export function useUpdateReminder() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateReminderRequest }) =>
      updateReminder(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reminders'] });
      queryClient.invalidateQueries({ queryKey: ['reminder'] });
    },
  });
}

export function useConfirmReminder() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => confirmReminder(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reminders'] });
      queryClient.invalidateQueries({ queryKey: ['reminder'] });
    },
  });
}

export function useRejectReminder() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => rejectReminder(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reminders'] });
      queryClient.invalidateQueries({ queryKey: ['reminder'] });
    },
  });
}

export function useCompleteReminder() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => completeReminder(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reminders'] });
      queryClient.invalidateQueries({ queryKey: ['reminder'] });
    },
  });
}

export function useDismissReminder() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => dismissReminder(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reminders'] });
      queryClient.invalidateQueries({ queryKey: ['reminder'] });
    },
  });
}
