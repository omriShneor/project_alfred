import React from 'react';
import { renderHook, waitFor, act } from '@testing-library/react-native';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  useEvents,
  useEvent,
  useUpdateEvent,
  useConfirmEvent,
  useRejectEvent,
  useChannelHistory,
  useCalendars,
} from '../../hooks/useEvents';
import * as eventsApi from '../../api/events';
import type { CalendarEvent, EventWithMessage, MessageHistory, Calendar } from '../../types/event';

// Mock the API module
jest.mock('../../api/events');

const mockEventsApi = eventsApi as jest.Mocked<typeof eventsApi>;

// Create a wrapper with QueryClientProvider
const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  });

  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

describe('useEvents hooks', () => {
  const mockEvent: CalendarEvent = {
    id: 1,
    channel_id: 1,
    channel_name: 'Test Channel',
    calendar_id: 'primary',
    title: 'Test Event',
    description: 'Test Description',
    start_time: '2024-01-15T10:00:00Z',
    end_time: '2024-01-15T11:00:00Z',
    location: 'Test Location',
    status: 'pending',
    action_type: 'create',
    llm_reasoning: 'AI detected this event',
    attendees: [{ id: 1, event_id: 1, name: 'John Doe', email: 'john@example.com' }],
    created_at: '2024-01-14T10:00:00Z',
    updated_at: '2024-01-14T10:00:00Z',
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('useEvents', () => {
    it('fetches events successfully', async () => {
      mockEventsApi.listEvents.mockResolvedValueOnce([mockEvent]);

      const { result } = renderHook(() => useEvents(), { wrapper: createWrapper() });

      expect(result.current.isLoading).toBe(true);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual([mockEvent]);
      expect(mockEventsApi.listEvents).toHaveBeenCalledWith(undefined);
    });

    it('fetches events with status filter', async () => {
      mockEventsApi.listEvents.mockResolvedValueOnce([mockEvent]);

      const { result } = renderHook(() => useEvents({ status: 'pending' }), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockEventsApi.listEvents).toHaveBeenCalledWith({ status: 'pending' });
    });

    it('fetches events with channel_id filter', async () => {
      mockEventsApi.listEvents.mockResolvedValueOnce([mockEvent]);

      const { result } = renderHook(() => useEvents({ channel_id: 1 }), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockEventsApi.listEvents).toHaveBeenCalledWith({ channel_id: 1 });
    });

    it('handles error state', async () => {
      mockEventsApi.listEvents.mockRejectedValueOnce(new Error('Network error'));

      const { result } = renderHook(() => useEvents(), { wrapper: createWrapper() });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });

      expect(result.current.error?.message).toBe('Network error');
    });

    it('returns empty array when no events', async () => {
      mockEventsApi.listEvents.mockResolvedValueOnce([]);

      const { result } = renderHook(() => useEvents(), { wrapper: createWrapper() });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual([]);
    });
  });

  describe('useEvent', () => {
    it('fetches single event when id > 0', async () => {
      const eventWithMessage: EventWithMessage = {
        ...mockEvent,
        trigger_message: {
          id: 1,
          channel_id: 1,
          sender_jid: '123@s.whatsapp.net',
          sender_name: 'John',
          message_text: 'Meeting tomorrow',
          timestamp: '2024-01-14T09:00:00Z',
        },
      };
      mockEventsApi.getEvent.mockResolvedValueOnce(eventWithMessage);

      const { result } = renderHook(() => useEvent(1), { wrapper: createWrapper() });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(eventWithMessage);
      expect(mockEventsApi.getEvent).toHaveBeenCalledWith(1);
    });

    it('does not fetch when id is 0', async () => {
      const { result } = renderHook(() => useEvent(0), { wrapper: createWrapper() });

      // Query should be disabled
      expect(result.current.fetchStatus).toBe('idle');
      expect(mockEventsApi.getEvent).not.toHaveBeenCalled();
    });

    it('does not fetch when id is negative', async () => {
      const { result } = renderHook(() => useEvent(-1), { wrapper: createWrapper() });

      expect(result.current.fetchStatus).toBe('idle');
      expect(mockEventsApi.getEvent).not.toHaveBeenCalled();
    });
  });

  describe('useUpdateEvent', () => {
    it('updates event successfully', async () => {
      const updatedEvent = { ...mockEvent, title: 'Updated Title' };
      mockEventsApi.updateEvent.mockResolvedValueOnce(updatedEvent);

      const { result } = renderHook(() => useUpdateEvent(), { wrapper: createWrapper() });

      await act(async () => {
        result.current.mutate({ id: 1, data: { title: 'Updated Title' } });
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockEventsApi.updateEvent).toHaveBeenCalledWith(1, { title: 'Updated Title' });
    });

    it('handles update error', async () => {
      mockEventsApi.updateEvent.mockRejectedValueOnce(new Error('Update failed'));

      const { result } = renderHook(() => useUpdateEvent(), { wrapper: createWrapper() });

      await act(async () => {
        result.current.mutate({ id: 1, data: { title: 'Updated Title' } });
      });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });

      expect(result.current.error?.message).toBe('Update failed');
    });
  });

  describe('useConfirmEvent', () => {
    it('confirms event successfully', async () => {
      const confirmedEvent = { ...mockEvent, status: 'confirmed' as const };
      mockEventsApi.confirmEvent.mockResolvedValueOnce(confirmedEvent);

      const { result } = renderHook(() => useConfirmEvent(), { wrapper: createWrapper() });

      await act(async () => {
        result.current.mutate(1);
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockEventsApi.confirmEvent).toHaveBeenCalledWith(1);
    });

    it('handles confirm error', async () => {
      mockEventsApi.confirmEvent.mockRejectedValueOnce(new Error('Confirm failed'));

      const { result } = renderHook(() => useConfirmEvent(), { wrapper: createWrapper() });

      await act(async () => {
        result.current.mutate(1);
      });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });
    });
  });

  describe('useRejectEvent', () => {
    it('rejects event successfully', async () => {
      const rejectedEvent = { ...mockEvent, status: 'rejected' as const };
      mockEventsApi.rejectEvent.mockResolvedValueOnce(rejectedEvent);

      const { result } = renderHook(() => useRejectEvent(), { wrapper: createWrapper() });

      await act(async () => {
        result.current.mutate(1);
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockEventsApi.rejectEvent).toHaveBeenCalledWith(1);
    });

    it('handles reject error', async () => {
      mockEventsApi.rejectEvent.mockRejectedValueOnce(new Error('Reject failed'));

      const { result } = renderHook(() => useRejectEvent(), { wrapper: createWrapper() });

      await act(async () => {
        result.current.mutate(1);
      });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });
    });
  });

  describe('useChannelHistory', () => {
    it('fetches channel history when channelId > 0', async () => {
      const mockHistory: MessageHistory[] = [
        {
          id: 1,
          channel_id: 1,
          sender_jid: '123@s.whatsapp.net',
          sender_name: 'John',
          message_text: 'Hello',
          timestamp: '2024-01-14T09:00:00Z',
        },
      ];
      mockEventsApi.getChannelHistory.mockResolvedValueOnce(mockHistory);

      const { result } = renderHook(() => useChannelHistory(1), { wrapper: createWrapper() });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockHistory);
      expect(mockEventsApi.getChannelHistory).toHaveBeenCalledWith(1);
    });

    it('does not fetch when channelId is 0', async () => {
      const { result } = renderHook(() => useChannelHistory(0), { wrapper: createWrapper() });

      expect(result.current.fetchStatus).toBe('idle');
      expect(mockEventsApi.getChannelHistory).not.toHaveBeenCalled();
    });
  });

  describe('useCalendars', () => {
    it('fetches calendars when enabled', async () => {
      const mockCalendars: Calendar[] = [
        { id: 'primary', summary: 'Primary', primary: true },
        { id: 'work', summary: 'Work', primary: false },
      ];
      mockEventsApi.listCalendars.mockResolvedValueOnce(mockCalendars);

      const { result } = renderHook(() => useCalendars(true), { wrapper: createWrapper() });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockCalendars);
    });

    it('does not fetch when disabled', async () => {
      const { result } = renderHook(() => useCalendars(false), { wrapper: createWrapper() });

      expect(result.current.fetchStatus).toBe('idle');
      expect(mockEventsApi.listCalendars).not.toHaveBeenCalled();
    });

    it('enabled by default', async () => {
      mockEventsApi.listCalendars.mockResolvedValueOnce([]);

      const { result } = renderHook(() => useCalendars(), { wrapper: createWrapper() });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockEventsApi.listCalendars).toHaveBeenCalled();
    });
  });
});
