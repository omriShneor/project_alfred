import React from 'react';
import { render, fireEvent, screen, waitFor } from '@testing-library/react-native';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Alert } from 'react-native';
import { EventCard } from '../../../components/events/EventCard';
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
      mutations: {
        retry: false,
      },
    },
  });

  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

describe('EventCard', () => {
  const mockEvent: CalendarEvent = {
    id: 1,
    channel_id: 1,
    channel_name: 'Family Group',
    calendar_id: 'primary',
    title: 'Family Dinner',
    description: 'Annual family gathering',
    start_time: '2024-01-15T18:00:00Z',
    end_time: '2024-01-15T21:00:00Z',
    location: 'Grandma\'s House',
    status: 'pending',
    action_type: 'create',
    llm_reasoning: 'Detected family dinner event from conversation about gathering next Sunday.',
    attendees: [
      { id: 1, event_id: 1, name: 'Mom', email: 'mom@family.com' },
      { id: 2, event_id: 1, name: 'Dad', email: 'dad@family.com' },
    ],
    created_at: '2024-01-14T10:00:00Z',
    updated_at: '2024-01-14T10:00:00Z',
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('rendering', () => {
    it('renders event title', () => {
      render(<EventCard event={mockEvent} />, { wrapper: createWrapper() });

      expect(screen.getByText('Family Dinner')).toBeTruthy();
    });

    it('renders action type badge', () => {
      render(<EventCard event={mockEvent} />, { wrapper: createWrapper() });

      expect(screen.getByText('create')).toBeTruthy();
    });

    it('renders status badge', () => {
      render(<EventCard event={mockEvent} />, { wrapper: createWrapper() });

      expect(screen.getByText('pending')).toBeTruthy();
    });

    it('renders channel name', () => {
      render(<EventCard event={mockEvent} />, { wrapper: createWrapper() });

      expect(screen.getByText('Family Group')).toBeTruthy();
    });

    it('renders date/time information', () => {
      render(<EventCard event={mockEvent} />, { wrapper: createWrapper() });

      expect(screen.getByText('When:')).toBeTruthy();
    });

    it('renders location when provided', () => {
      render(<EventCard event={mockEvent} />, { wrapper: createWrapper() });

      expect(screen.getByText('Where:')).toBeTruthy();
      expect(screen.getByText("Grandma's House")).toBeTruthy();
    });

    it('does not render location label when not provided', () => {
      const eventWithoutLocation = { ...mockEvent, location: '' };
      render(<EventCard event={eventWithoutLocation} />, { wrapper: createWrapper() });

      expect(screen.queryByText('Where:')).toBeNull();
    });

    it('renders attendees when present', () => {
      render(<EventCard event={mockEvent} />, { wrapper: createWrapper() });

      expect(screen.getByText('Mom')).toBeTruthy();
      expect(screen.getByText('Dad')).toBeTruthy();
    });
  });

  describe('AI reasoning', () => {
    it('renders AI reasoning toggle', () => {
      render(<EventCard event={mockEvent} />, { wrapper: createWrapper() });

      expect(screen.getByText(/AI Reasoning/)).toBeTruthy();
    });

    it('shows AI reasoning when toggle is pressed', () => {
      render(<EventCard event={mockEvent} />, { wrapper: createWrapper() });

      fireEvent.press(screen.getByText(/AI Reasoning/));

      expect(screen.getByText(/Detected family dinner event/)).toBeTruthy();
    });

    it('hides AI reasoning when toggle is pressed again', () => {
      render(<EventCard event={mockEvent} />, { wrapper: createWrapper() });

      // Show reasoning
      fireEvent.press(screen.getByText(/AI Reasoning/));
      expect(screen.getByText(/Detected family dinner event/)).toBeTruthy();

      // Hide reasoning
      fireEvent.press(screen.getByText(/AI Reasoning/));
      expect(screen.queryByText(/Detected family dinner event/)).toBeNull();
    });

    it('does not render AI reasoning toggle when reasoning is not provided', () => {
      const eventWithoutReasoning = { ...mockEvent, llm_reasoning: '' };
      render(<EventCard event={eventWithoutReasoning} />, { wrapper: createWrapper() });

      // The toggle text should not be present
      expect(screen.queryByText('AI Reasoning ▼')).toBeNull();
    });
  });

  describe('action buttons', () => {
    it('renders View Context button', () => {
      render(<EventCard event={mockEvent} />, { wrapper: createWrapper() });

      expect(screen.getByText('View Context')).toBeTruthy();
    });

    it('renders Edit button for pending events', () => {
      render(<EventCard event={mockEvent} />, { wrapper: createWrapper() });

      expect(screen.getByText('Edit')).toBeTruthy();
    });

    it('renders Confirm button for pending events', () => {
      render(<EventCard event={mockEvent} />, { wrapper: createWrapper() });

      expect(screen.getByText('Confirm')).toBeTruthy();
    });

    it('renders Reject button for pending events', () => {
      render(<EventCard event={mockEvent} />, { wrapper: createWrapper() });

      expect(screen.getByText('Reject')).toBeTruthy();
    });

    it('does not render Edit/Confirm/Reject for confirmed events', () => {
      const confirmedEvent = { ...mockEvent, status: 'confirmed' as const };
      render(<EventCard event={confirmedEvent} />, { wrapper: createWrapper() });

      expect(screen.queryByText('Edit')).toBeNull();
      expect(screen.queryByText('Confirm')).toBeNull();
      expect(screen.queryByText('Reject')).toBeNull();
    });

    it('does not render Edit/Confirm/Reject for synced events', () => {
      const syncedEvent = { ...mockEvent, status: 'synced' as const };
      render(<EventCard event={syncedEvent} />, { wrapper: createWrapper() });

      expect(screen.queryByText('Edit')).toBeNull();
      expect(screen.queryByText('Confirm')).toBeNull();
      expect(screen.queryByText('Reject')).toBeNull();
    });

    it('does not render Edit/Confirm/Reject for rejected events', () => {
      const rejectedEvent = { ...mockEvent, status: 'rejected' as const };
      render(<EventCard event={rejectedEvent} />, { wrapper: createWrapper() });

      expect(screen.queryByText('Edit')).toBeNull();
      expect(screen.queryByText('Confirm')).toBeNull();
      expect(screen.queryByText('Reject')).toBeNull();
    });
  });

  describe('confirm event flow', () => {
    it('shows confirmation alert when Confirm is pressed', () => {
      const alertSpy = jest.spyOn(Alert, 'alert');
      render(<EventCard event={mockEvent} />, { wrapper: createWrapper() });

      fireEvent.press(screen.getByText('Confirm'));

      expect(alertSpy).toHaveBeenCalledWith(
        'Confirm Event',
        'Sync this event to Google Calendar?',
        expect.any(Array)
      );
    });

    it('calls confirmEvent when confirmed in alert', async () => {
      const confirmedEvent = { ...mockEvent, status: 'confirmed' as const };
      mockEventsApi.confirmEvent.mockResolvedValueOnce(confirmedEvent);

      const alertSpy = jest.spyOn(Alert, 'alert');
      render(<EventCard event={mockEvent} />, { wrapper: createWrapper() });

      fireEvent.press(screen.getByText('Confirm'));

      // Get the alert callback and simulate pressing Confirm
      const alertButtons = alertSpy.mock.calls[0][2] as any[];
      const confirmButton = alertButtons.find((b: any) => b.text === 'Confirm');
      confirmButton.onPress();

      await waitFor(() => {
        expect(mockEventsApi.confirmEvent).toHaveBeenCalledWith(mockEvent.id);
      });
    });
  });

  describe('reject event flow', () => {
    it('shows rejection alert when Reject is pressed', () => {
      const alertSpy = jest.spyOn(Alert, 'alert');
      render(<EventCard event={mockEvent} />, { wrapper: createWrapper() });

      fireEvent.press(screen.getByText('Reject'));

      expect(alertSpy).toHaveBeenCalledWith(
        'Reject Event',
        'Are you sure you want to reject this event?',
        expect.any(Array)
      );
    });

    it('calls rejectEvent when rejected in alert', async () => {
      const rejectedEvent = { ...mockEvent, status: 'rejected' as const };
      mockEventsApi.rejectEvent.mockResolvedValueOnce(rejectedEvent);

      const alertSpy = jest.spyOn(Alert, 'alert');
      render(<EventCard event={mockEvent} />, { wrapper: createWrapper() });

      fireEvent.press(screen.getByText('Reject'));

      // Get the alert callback and simulate pressing Reject
      const alertButtons = alertSpy.mock.calls[0][2] as any[];
      const rejectButton = alertButtons.find((b: any) => b.text === 'Reject');
      rejectButton.onPress();

      await waitFor(() => {
        expect(mockEventsApi.rejectEvent).toHaveBeenCalledWith(mockEvent.id);
      });
    });
  });

  describe('different action types', () => {
    it('renders update action type badge', () => {
      const updateEvent = { ...mockEvent, action_type: 'update' as const };
      render(<EventCard event={updateEvent} />, { wrapper: createWrapper() });

      expect(screen.getByText('update')).toBeTruthy();
    });

    it('renders delete action type badge', () => {
      const deleteEvent = { ...mockEvent, action_type: 'delete' as const };
      render(<EventCard event={deleteEvent} />, { wrapper: createWrapper() });

      expect(screen.getByText('delete')).toBeTruthy();
    });
  });

  describe('different statuses', () => {
    const statuses = ['pending', 'confirmed', 'synced', 'rejected'] as const;

    statuses.forEach((status) => {
      it(`renders ${status} status badge`, () => {
        const eventWithStatus = { ...mockEvent, status };
        render(<EventCard event={eventWithStatus} />, { wrapper: createWrapper() });

        expect(screen.getByText(status)).toBeTruthy();
      });
    });
  });

  describe('edge cases', () => {
    it('handles event without channel name', () => {
      const eventWithoutChannel = { ...mockEvent, channel_name: undefined };
      render(<EventCard event={eventWithoutChannel} />, { wrapper: createWrapper() });

      expect(screen.getByText('Family Dinner')).toBeTruthy();
    });

    it('handles event without end time', () => {
      const eventWithoutEndTime = { ...mockEvent, end_time: undefined };
      render(<EventCard event={eventWithoutEndTime} />, { wrapper: createWrapper() });

      expect(screen.getByText('When:')).toBeTruthy();
    });

    it('handles event without attendees', () => {
      const eventWithoutAttendees = { ...mockEvent, attendees: [] };
      render(<EventCard event={eventWithoutAttendees} />, { wrapper: createWrapper() });

      expect(screen.queryByText('Attendees:')).toBeNull();
    });

    it('handles long title', () => {
      const eventWithLongTitle = {
        ...mockEvent,
        title: 'This is a very long event title that might need to be truncated or wrapped',
      };
      render(<EventCard event={eventWithLongTitle} />, { wrapper: createWrapper() });

      expect(screen.getByText(eventWithLongTitle.title)).toBeTruthy();
    });
  });
});
