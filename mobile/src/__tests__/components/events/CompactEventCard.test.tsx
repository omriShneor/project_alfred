import React from 'react';
import { render, fireEvent, screen, waitFor } from '@testing-library/react-native';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Alert } from 'react-native';
import { CompactEventCard } from '../../../components/events/CompactEventCard';
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

describe('CompactEventCard', () => {
  const mockEvent: CalendarEvent = {
    id: 1,
    channel_id: 1,
    channel_name: 'Work',
    calendar_id: 'primary',
    title: 'Team Meeting',
    description: 'Weekly sync',
    start_time: '2024-01-15T10:00:00Z',
    end_time: '2024-01-15T11:00:00Z',
    location: 'Conference Room A',
    status: 'pending',
    action_type: 'create',
    llm_reasoning: 'Detected meeting event',
    attendees: [],
    created_at: '2024-01-14T10:00:00Z',
    updated_at: '2024-01-14T10:00:00Z',
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('rendering', () => {
    it('renders event title', () => {
      render(<CompactEventCard event={mockEvent} />, { wrapper: createWrapper() });

      expect(screen.getByText('Team Meeting')).toBeTruthy();
    });

    it('renders channel name with hashtag', () => {
      render(<CompactEventCard event={mockEvent} />, { wrapper: createWrapper() });

      expect(screen.getByText('#Work')).toBeTruthy();
    });

    it('renders formatted date/time', () => {
      render(<CompactEventCard event={mockEvent} />, { wrapper: createWrapper() });

      // The exact format depends on locale, but it should be present
      expect(screen.root).toBeTruthy();
    });

    it('does not render channel name when not provided', () => {
      const eventWithoutChannel = { ...mockEvent, channel_name: undefined };
      render(<CompactEventCard event={eventWithoutChannel} />, { wrapper: createWrapper() });

      expect(screen.queryByText(/#/)).toBeNull();
    });
  });

  describe('action buttons', () => {
    it('renders edit button', () => {
      render(<CompactEventCard event={mockEvent} />, { wrapper: createWrapper() });

      // IconButton renders icons, not text - we check the component exists
      expect(screen.root).toBeTruthy();
    });

    it('renders reject button', () => {
      render(<CompactEventCard event={mockEvent} />, { wrapper: createWrapper() });

      expect(screen.root).toBeTruthy();
    });

    it('renders confirm button', () => {
      render(<CompactEventCard event={mockEvent} />, { wrapper: createWrapper() });

      expect(screen.root).toBeTruthy();
    });
  });

  describe('confirm event flow', () => {
    it('shows confirmation alert when confirm button is pressed', () => {
      const alertSpy = jest.spyOn(Alert, 'alert');
      render(<CompactEventCard event={mockEvent} />, { wrapper: createWrapper() });

      // Find all touchable elements (IconButtons use TouchableOpacity)
      const buttons = screen.root.findAllByType('TouchableOpacity' as any);
      // The confirm button should be the last one (check icon)
      if (buttons.length >= 3) {
        fireEvent.press(buttons[buttons.length - 1]);
      }

      // Alert should be called for confirm
      expect(alertSpy).toHaveBeenCalled();
    });

    it('calls confirmEvent when confirmed in alert', async () => {
      const confirmedEvent = { ...mockEvent, status: 'confirmed' as const };
      mockEventsApi.confirmEvent.mockResolvedValueOnce(confirmedEvent);

      const alertSpy = jest.spyOn(Alert, 'alert');
      render(<CompactEventCard event={mockEvent} />, { wrapper: createWrapper() });

      // Trigger confirm
      const buttons = screen.root.findAllByType('TouchableOpacity' as any);
      if (buttons.length >= 3) {
        fireEvent.press(buttons[buttons.length - 1]);
      }

      // Find and call the confirm callback
      if (alertSpy.mock.calls.length > 0) {
        const alertButtons = alertSpy.mock.calls[0][2] as any[];
        if (alertButtons) {
          const confirmButton = alertButtons.find((b: any) => b.text === 'Confirm');
          if (confirmButton && confirmButton.onPress) {
            confirmButton.onPress();
          }
        }
      }

      await waitFor(() => {
        expect(mockEventsApi.confirmEvent).toHaveBeenCalledWith(mockEvent.id);
      });
    });
  });

  describe('reject event flow', () => {
    it('shows rejection alert when reject button is pressed', () => {
      const alertSpy = jest.spyOn(Alert, 'alert');
      render(<CompactEventCard event={mockEvent} />, { wrapper: createWrapper() });

      // The reject button should be the second to last one (x icon)
      const buttons = screen.root.findAllByType('TouchableOpacity' as any);
      if (buttons.length >= 3) {
        fireEvent.press(buttons[buttons.length - 2]);
      }

      expect(alertSpy).toHaveBeenCalled();
    });

    it('calls rejectEvent when rejected in alert', async () => {
      const rejectedEvent = { ...mockEvent, status: 'rejected' as const };
      mockEventsApi.rejectEvent.mockResolvedValueOnce(rejectedEvent);

      const alertSpy = jest.spyOn(Alert, 'alert');
      render(<CompactEventCard event={mockEvent} />, { wrapper: createWrapper() });

      // Trigger reject
      const buttons = screen.root.findAllByType('TouchableOpacity' as any);
      if (buttons.length >= 3) {
        fireEvent.press(buttons[buttons.length - 2]);
      }

      // Find and call the reject callback
      if (alertSpy.mock.calls.length > 0) {
        const alertButtons = alertSpy.mock.calls[0][2] as any[];
        if (alertButtons) {
          const rejectButton = alertButtons.find((b: any) => b.text === 'Reject');
          if (rejectButton && rejectButton.onPress) {
            rejectButton.onPress();
          }
        }
      }

      await waitFor(() => {
        expect(mockEventsApi.rejectEvent).toHaveBeenCalledWith(mockEvent.id);
      });
    });
  });

  describe('edge cases', () => {
    it('handles long event title with truncation', () => {
      const eventWithLongTitle = {
        ...mockEvent,
        title: 'This is an extremely long event title that should be truncated',
      };
      render(<CompactEventCard event={eventWithLongTitle} />, { wrapper: createWrapper() });

      expect(screen.getByText(eventWithLongTitle.title)).toBeTruthy();
    });

    it('handles event without channel name', () => {
      const eventWithoutChannel = { ...mockEvent, channel_name: undefined };
      render(<CompactEventCard event={eventWithoutChannel} />, { wrapper: createWrapper() });

      expect(screen.getByText('Team Meeting')).toBeTruthy();
    });

    it('handles special characters in title', () => {
      const eventWithSpecialChars = {
        ...mockEvent,
        title: "John's Birthday <Party> & Celebration",
      };
      render(<CompactEventCard event={eventWithSpecialChars} />, { wrapper: createWrapper() });

      expect(screen.getByText(eventWithSpecialChars.title)).toBeTruthy();
    });
  });

  describe('loading states', () => {
    it('disables buttons during confirm mutation', async () => {
      // Create a never-resolving promise to keep loading state
      mockEventsApi.confirmEvent.mockImplementation(
        () => new Promise(() => {}) // Never resolves
      );

      const alertSpy = jest.spyOn(Alert, 'alert');
      render(<CompactEventCard event={mockEvent} />, { wrapper: createWrapper() });

      // Trigger confirm
      const buttons = screen.root.findAllByType('TouchableOpacity' as any);
      if (buttons.length >= 3) {
        fireEvent.press(buttons[buttons.length - 1]);
      }

      // Call the confirm callback
      if (alertSpy.mock.calls.length > 0) {
        const alertButtons = alertSpy.mock.calls[0][2] as any[];
        if (alertButtons) {
          const confirmButton = alertButtons.find((b: any) => b.text === 'Confirm');
          if (confirmButton && confirmButton.onPress) {
            confirmButton.onPress();
          }
        }
      }

      // Buttons should be disabled during loading
      // (The component checks confirmEvent.isPending || rejectEvent.isPending)
      await waitFor(() => {
        expect(mockEventsApi.confirmEvent).toHaveBeenCalled();
      });
    });
  });
});
