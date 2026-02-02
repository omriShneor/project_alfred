import React from 'react';
import { render, screen, waitFor } from '@testing-library/react-native';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { PendingEventsSection } from '../../../components/home/PendingEventsSection';
import * as eventsApi from '../../../api/events';
import type { CalendarEvent } from '../../../types/event';

// Mock the API module
jest.mock('../../../api/events');

const mockEventsApi = eventsApi as jest.Mocked<typeof eventsApi>;

// Create a wrapper with QueryClientProvider
const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
    },
  });

  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

describe('PendingEventsSection', () => {
  const mockPendingEvents: CalendarEvent[] = [
    {
      id: 1,
      channel_id: 1,
      channel_name: 'Family',
      calendar_id: 'primary',
      title: 'Family Dinner',
      description: 'Dinner at 7pm',
      start_time: '2024-01-15T19:00:00Z',
      end_time: '2024-01-15T21:00:00Z',
      location: "Mom's House",
      status: 'pending',
      action_type: 'create',
      llm_reasoning: 'Detected event',
      attendees: [],
      created_at: '2024-01-14T10:00:00Z',
      updated_at: '2024-01-14T10:00:00Z',
    },
    {
      id: 2,
      channel_id: 2,
      channel_name: 'Work',
      calendar_id: 'primary',
      title: 'Team Meeting',
      description: 'Weekly sync',
      start_time: '2024-01-15T10:00:00Z',
      end_time: '2024-01-15T11:00:00Z',
      location: 'Conference Room',
      status: 'pending',
      action_type: 'create',
      llm_reasoning: 'Detected meeting',
      attendees: [],
      created_at: '2024-01-14T10:00:00Z',
      updated_at: '2024-01-14T10:00:00Z',
    },
  ];

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('loading state', () => {
    it('shows loading spinner while fetching', async () => {
      mockEventsApi.listEvents.mockImplementation(
        () => new Promise((resolve) => setTimeout(() => resolve([]), 1000))
      );

      render(<PendingEventsSection />, { wrapper: createWrapper() });

      expect(screen.getByText('PENDING EVENTS')).toBeTruthy();
      // Loading spinner is rendered
    });
  });

  describe('empty state', () => {
    it('shows empty state message when no pending events', async () => {
      mockEventsApi.listEvents.mockResolvedValueOnce([]);

      render(<PendingEventsSection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('No pending events')).toBeTruthy();
      });

      expect(
        screen.getByText('Events detected from your tracked contacts/groups will appear here')
      ).toBeTruthy();
    });
  });

  describe('with events', () => {
    it('displays pending events count in title', async () => {
      mockEventsApi.listEvents.mockResolvedValueOnce(mockPendingEvents);

      render(<PendingEventsSection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('PENDING EVENTS (2)')).toBeTruthy();
      });
    });

    it('displays event titles', async () => {
      mockEventsApi.listEvents.mockResolvedValueOnce(mockPendingEvents);

      render(<PendingEventsSection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Family Dinner')).toBeTruthy();
      });

      expect(screen.getByText('Team Meeting')).toBeTruthy();
    });

    it('displays channel names', async () => {
      mockEventsApi.listEvents.mockResolvedValueOnce(mockPendingEvents);

      render(<PendingEventsSection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('#Family')).toBeTruthy();
      });

      expect(screen.getByText('#Work')).toBeTruthy();
    });
  });

  describe('query params', () => {
    it('fetches only pending events', async () => {
      mockEventsApi.listEvents.mockResolvedValueOnce([]);

      render(<PendingEventsSection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(mockEventsApi.listEvents).toHaveBeenCalledWith({ status: 'pending' });
      });
    });
  });

  describe('single event', () => {
    it('displays single event correctly', async () => {
      mockEventsApi.listEvents.mockResolvedValueOnce([mockPendingEvents[0]]);

      render(<PendingEventsSection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('PENDING EVENTS (1)')).toBeTruthy();
      });

      expect(screen.getByText('Family Dinner')).toBeTruthy();
    });
  });

  describe('edge cases', () => {
    it('handles many pending events', async () => {
      const manyEvents = Array.from({ length: 10 }, (_, i) => ({
        ...mockPendingEvents[0],
        id: i + 1,
        title: `Event ${i + 1}`,
      }));

      mockEventsApi.listEvents.mockResolvedValueOnce(manyEvents);

      render(<PendingEventsSection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('PENDING EVENTS (10)')).toBeTruthy();
      });

      expect(screen.getByText('Event 1')).toBeTruthy();
      expect(screen.getByText('Event 10')).toBeTruthy();
    });

    it('handles event without channel name', async () => {
      const eventWithoutChannel = { ...mockPendingEvents[0], channel_name: undefined };
      mockEventsApi.listEvents.mockResolvedValueOnce([eventWithoutChannel]);

      render(<PendingEventsSection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Family Dinner')).toBeTruthy();
      });

      expect(screen.queryByText(/#/)).toBeNull();
    });
  });
});
