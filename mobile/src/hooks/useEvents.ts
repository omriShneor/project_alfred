import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  listEvents,
  getEvent,
  updateEvent,
  confirmEvent,
  rejectEvent,
  getChannelHistory,
  listCalendars,
  type ListEventsParams,
} from '../api/events';
import type {
  CalendarEvent,
  EventWithMessage,
  UpdateEventRequest,
  MessageHistory,
  Calendar,
} from '../types/event';

export function useEvents(params?: ListEventsParams) {
  return useQuery<CalendarEvent[]>({
    queryKey: ['events', params],
    queryFn: () => listEvents(params),
  });
}

export function useEvent(id: number) {
  return useQuery<EventWithMessage>({
    queryKey: ['event', id],
    queryFn: () => getEvent(id),
    enabled: id > 0,
  });
}

export function useUpdateEvent() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateEventRequest }) =>
      updateEvent(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['events'] });
      queryClient.invalidateQueries({ queryKey: ['event'] });
    },
  });
}

export function useConfirmEvent() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => confirmEvent(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['events'] });
      queryClient.invalidateQueries({ queryKey: ['event'] });
    },
  });
}

export function useRejectEvent() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => rejectEvent(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['events'] });
      queryClient.invalidateQueries({ queryKey: ['event'] });
    },
  });
}

export function useChannelHistory(channelId: number) {
  return useQuery<MessageHistory[]>({
    queryKey: ['channelHistory', channelId],
    queryFn: () => getChannelHistory(channelId),
    enabled: channelId > 0,
  });
}

export function useCalendars() {
  return useQuery<Calendar[]>({
    queryKey: ['calendars'],
    queryFn: listCalendars,
  });
}
