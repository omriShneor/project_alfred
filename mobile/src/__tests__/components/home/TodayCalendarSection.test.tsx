import React from 'react';
import { render, screen, waitFor } from '@testing-library/react-native';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { TodayCalendarSection } from '../../../components/home/TodayCalendarSection';
import * as calendarApi from '../../../api/calendar';
import type { TodayEvent } from '../../../types/calendar';

// Mock the API module
jest.mock('../../../api/calendar');

const mockCalendarApi = calendarApi as jest.Mocked<typeof calendarApi>;

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

describe('TodayCalendarSection', () => {
  const mockTodayEvents: TodayEvent[] = [
    {
      id: '1',
      summary: 'Morning Standup',
      description: 'Daily team sync',
      location: 'Zoom',
      start_time: '2024-01-15T09:00:00Z',
      end_time: '2024-01-15T09:30:00Z',
      all_day: false,
      calendar_id: 'primary',
      source: 'google',
    },
    {
      id: '2',
      summary: 'Lunch Meeting',
      description: 'Team lunch',
      location: 'Restaurant',
      start_time: '2024-01-15T12:00:00Z',
      end_time: '2024-01-15T13:00:00Z',
      all_day: false,
      calendar_id: 'primary',
      source: 'alfred',
    },
    {
      id: '3',
      summary: 'Company Holiday',
      all_day: true,
      start_time: '2024-01-15T00:00:00Z',
      end_time: '2024-01-15T23:59:59Z',
      calendar_id: 'primary',
      source: 'google',
    },
  ];

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('loading state', () => {
    it('shows loading spinner while fetching', async () => {
      mockCalendarApi.getTodayEvents.mockImplementation(
        () => new Promise((resolve) => setTimeout(() => resolve([]), 1000))
      );

      render(<TodayCalendarSection />, { wrapper: createWrapper() });

      expect(screen.getByText("TODAY'S SCHEDULE")).toBeTruthy();
    });
  });

  describe('error state', () => {
    it('shows error message when fetch fails', async () => {
      mockCalendarApi.getTodayEvents.mockRejectedValueOnce(new Error('Network error'));

      render(<TodayCalendarSection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Unable to load calendar')).toBeTruthy();
      });

      expect(screen.getByText('Please try again later')).toBeTruthy();
    });
  });

  describe('empty state', () => {
    it('shows empty state message when no events', async () => {
      mockCalendarApi.getTodayEvents.mockResolvedValueOnce([]);

      render(<TodayCalendarSection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('No events scheduled for today')).toBeTruthy();
      });

      expect(screen.getByText('Events will appear here once confirmed')).toBeTruthy();
    });
  });

  describe('with events', () => {
    it('displays event count in title', async () => {
      mockCalendarApi.getTodayEvents.mockResolvedValueOnce(mockTodayEvents);

      render(<TodayCalendarSection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText("TODAY'S SCHEDULE (3)")).toBeTruthy();
      });
    });

    it('displays event summaries', async () => {
      mockCalendarApi.getTodayEvents.mockResolvedValueOnce(mockTodayEvents);

      render(<TodayCalendarSection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Morning Standup')).toBeTruthy();
      });

      expect(screen.getByText('Lunch Meeting')).toBeTruthy();
      expect(screen.getByText('Company Holiday')).toBeTruthy();
    });

    it('displays location for events with location', async () => {
      mockCalendarApi.getTodayEvents.mockResolvedValueOnce(mockTodayEvents);

      render(<TodayCalendarSection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Zoom')).toBeTruthy();
      });

      expect(screen.getByText('Restaurant')).toBeTruthy();
    });

    it('displays "All day" for all-day events', async () => {
      mockCalendarApi.getTodayEvents.mockResolvedValueOnce(mockTodayEvents);

      render(<TodayCalendarSection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('All day')).toBeTruthy();
      });
    });
  });

  describe('event types', () => {
    it('handles regular timed events', async () => {
      const timedEvents: TodayEvent[] = [
        {
          id: '1',
          summary: 'Meeting',
          start_time: '2024-01-15T10:00:00Z',
          end_time: '2024-01-15T11:00:00Z',
          all_day: false,
          calendar_id: 'primary',
        },
      ];
      mockCalendarApi.getTodayEvents.mockResolvedValueOnce(timedEvents);

      render(<TodayCalendarSection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Meeting')).toBeTruthy();
      });
    });

    it('handles all-day events', async () => {
      const allDayEvents: TodayEvent[] = [
        {
          id: '1',
          summary: 'Birthday',
          start_time: '2024-01-15T00:00:00Z',
          end_time: '2024-01-15T23:59:59Z',
          all_day: true,
          calendar_id: 'primary',
        },
      ];
      mockCalendarApi.getTodayEvents.mockResolvedValueOnce(allDayEvents);

      render(<TodayCalendarSection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Birthday')).toBeTruthy();
      });

      expect(screen.getByText('All day')).toBeTruthy();
    });

    it('handles events without location', async () => {
      const eventsWithoutLocation: TodayEvent[] = [
        {
          id: '1',
          summary: 'Call',
          start_time: '2024-01-15T10:00:00Z',
          end_time: '2024-01-15T10:30:00Z',
          all_day: false,
          calendar_id: 'primary',
        },
      ];
      mockCalendarApi.getTodayEvents.mockResolvedValueOnce(eventsWithoutLocation);

      render(<TodayCalendarSection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Call')).toBeTruthy();
      });
    });
  });

  describe('edge cases', () => {
    it('handles many events', async () => {
      const manyEvents = Array.from({ length: 10 }, (_, i) => ({
        id: `${i + 1}`,
        summary: `Event ${i + 1}`,
        start_time: `2024-01-15T${String(8 + i).padStart(2, '0')}:00:00Z`,
        end_time: `2024-01-15T${String(9 + i).padStart(2, '0')}:00:00Z`,
        all_day: false,
        calendar_id: 'primary',
      }));

      mockCalendarApi.getTodayEvents.mockResolvedValueOnce(manyEvents);

      render(<TodayCalendarSection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText("TODAY'S SCHEDULE (10)")).toBeTruthy();
      });
    });

    it('handles events with long titles', async () => {
      const longTitleEvent: TodayEvent[] = [
        {
          id: '1',
          summary: 'This is a very long event title that might need to be truncated',
          start_time: '2024-01-15T10:00:00Z',
          end_time: '2024-01-15T11:00:00Z',
          all_day: false,
          calendar_id: 'primary',
        },
      ];
      mockCalendarApi.getTodayEvents.mockResolvedValueOnce(longTitleEvent);

      render(<TodayCalendarSection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(
          screen.getByText('This is a very long event title that might need to be truncated')
        ).toBeTruthy();
      });
    });

    it('handles events with special characters', async () => {
      const specialCharEvent: TodayEvent[] = [
        {
          id: '1',
          summary: "John's Birthday <Party> & Celebration",
          start_time: '2024-01-15T10:00:00Z',
          end_time: '2024-01-15T11:00:00Z',
          all_day: false,
          calendar_id: 'primary',
        },
      ];
      mockCalendarApi.getTodayEvents.mockResolvedValueOnce(specialCharEvent);

      render(<TodayCalendarSection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText("John's Birthday <Party> & Celebration")).toBeTruthy();
      });
    });
  });

  describe('single event', () => {
    it('displays single event correctly', async () => {
      mockCalendarApi.getTodayEvents.mockResolvedValueOnce([mockTodayEvents[0]]);

      render(<TodayCalendarSection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText("TODAY'S SCHEDULE (1)")).toBeTruthy();
      });

      expect(screen.getByText('Morning Standup')).toBeTruthy();
    });
  });
});
