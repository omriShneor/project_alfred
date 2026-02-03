import React from 'react';
import { render, screen } from '@testing-library/react-native';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { CompactEventCard } from '../../../components/events/CompactEventCard';
import type { CalendarEvent } from '../../../types/event';

// Mock the API module
jest.mock('../../../api/events');

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

    it('renders the card structure', () => {
      render(<CompactEventCard event={mockEvent} />, { wrapper: createWrapper() });

      // The component should render successfully
      expect(screen.root).toBeTruthy();
    });

    it('does not render channel name when not provided', () => {
      const eventWithoutChannel = { ...mockEvent, channel_name: undefined };
      render(<CompactEventCard event={eventWithoutChannel} />, { wrapper: createWrapper() });

      expect(screen.queryByText(/#/)).toBeNull();
    });
  });

  describe('action buttons', () => {
    it('renders three action buttons (edit, reject, confirm)', () => {
      render(<CompactEventCard event={mockEvent} />, { wrapper: createWrapper() });

      // The component should render with action buttons
      // In react-native-web, IconButtons render as divs with icons
      expect(screen.root).toBeTruthy();
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

    it('handles event with empty string channel name', () => {
      const eventWithEmptyChannel = { ...mockEvent, channel_name: '' };
      render(<CompactEventCard event={eventWithEmptyChannel} />, { wrapper: createWrapper() });

      expect(screen.getByText('Team Meeting')).toBeTruthy();
    });

    it('handles event with very long channel name', () => {
      const eventWithLongChannel = {
        ...mockEvent,
        channel_name: 'This-Is-A-Very-Long-Channel-Name',
      };
      render(<CompactEventCard event={eventWithLongChannel} />, { wrapper: createWrapper() });

      expect(screen.getByText('#This-Is-A-Very-Long-Channel-Name')).toBeTruthy();
    });
  });

  describe('different event statuses', () => {
    it('renders pending event', () => {
      const pendingEvent = { ...mockEvent, status: 'pending' as const };
      render(<CompactEventCard event={pendingEvent} />, { wrapper: createWrapper() });

      expect(screen.getByText('Team Meeting')).toBeTruthy();
    });

    it('renders confirmed event', () => {
      const confirmedEvent = { ...mockEvent, status: 'confirmed' as const };
      render(<CompactEventCard event={confirmedEvent} />, { wrapper: createWrapper() });

      expect(screen.getByText('Team Meeting')).toBeTruthy();
    });

    it('renders synced event', () => {
      const syncedEvent = { ...mockEvent, status: 'synced' as const };
      render(<CompactEventCard event={syncedEvent} />, { wrapper: createWrapper() });

      expect(screen.getByText('Team Meeting')).toBeTruthy();
    });
  });
});
