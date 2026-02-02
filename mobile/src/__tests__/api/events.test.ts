import {
  listEvents,
  getEvent,
  updateEvent,
  confirmEvent,
  rejectEvent,
  getChannelHistory,
  listCalendars,
} from '../../api/events';
import { apiClient } from '../../api/client';
import type { CalendarEvent, EventWithMessage, MessageHistory, Calendar } from '../../types/event';

// Mock the API client
jest.mock('../../api/client', () => ({
  apiClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
  },
}));

const mockApiClient = apiClient as jest.Mocked<typeof apiClient>;

describe('Events API', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

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
    attendees: [],
    created_at: '2024-01-14T10:00:00Z',
    updated_at: '2024-01-14T10:00:00Z',
  };

  describe('listEvents', () => {
    it('fetches events without params', async () => {
      const mockEvents = [mockEvent];
      mockApiClient.get.mockResolvedValueOnce(mockEvents);

      const result = await listEvents();

      expect(mockApiClient.get).toHaveBeenCalledWith('/api/events', { params: undefined });
      expect(result).toEqual(mockEvents);
    });

    it('fetches events with status filter', async () => {
      const mockEvents = [mockEvent];
      mockApiClient.get.mockResolvedValueOnce(mockEvents);

      const result = await listEvents({ status: 'pending' });

      expect(mockApiClient.get).toHaveBeenCalledWith('/api/events', {
        params: { status: 'pending' },
      });
      expect(result).toEqual(mockEvents);
    });

    it('fetches events with channel_id filter', async () => {
      const mockEvents = [mockEvent];
      mockApiClient.get.mockResolvedValueOnce(mockEvents);

      const result = await listEvents({ channel_id: 123 });

      expect(mockApiClient.get).toHaveBeenCalledWith('/api/events', {
        params: { channel_id: 123 },
      });
      expect(result).toEqual(mockEvents);
    });

    it('fetches events with multiple filters', async () => {
      const mockEvents = [mockEvent];
      mockApiClient.get.mockResolvedValueOnce(mockEvents);

      const result = await listEvents({ status: 'confirmed', channel_id: 456 });

      expect(mockApiClient.get).toHaveBeenCalledWith('/api/events', {
        params: { status: 'confirmed', channel_id: 456 },
      });
      expect(result).toEqual(mockEvents);
    });

    it('returns empty array when no events', async () => {
      mockApiClient.get.mockResolvedValueOnce([]);

      const result = await listEvents();

      expect(result).toEqual([]);
    });
  });

  describe('getEvent', () => {
    it('fetches a single event by ID', async () => {
      const mockEventWithMessage: EventWithMessage = {
        ...mockEvent,
        trigger_message: {
          id: 1,
          channel_id: 1,
          sender_jid: '1234567890@s.whatsapp.net',
          sender_name: 'John Doe',
          message_text: 'Meeting tomorrow at 10am',
          timestamp: '2024-01-14T09:00:00Z',
        },
      };
      mockApiClient.get.mockResolvedValueOnce(mockEventWithMessage);

      const result = await getEvent(1);

      expect(mockApiClient.get).toHaveBeenCalledWith('/api/events/1');
      expect(result).toEqual(mockEventWithMessage);
    });

    it('fetches event without trigger message', async () => {
      const mockEventWithMessage: EventWithMessage = {
        ...mockEvent,
        trigger_message: undefined,
      };
      mockApiClient.get.mockResolvedValueOnce(mockEventWithMessage);

      const result = await getEvent(1);

      expect(result.trigger_message).toBeUndefined();
    });
  });

  describe('updateEvent', () => {
    it('updates event with all fields', async () => {
      const updateData = {
        title: 'Updated Title',
        description: 'Updated Description',
        start_time: '2024-01-15T14:00:00Z',
        end_time: '2024-01-15T15:00:00Z',
        location: 'New Location',
        attendees: [{ name: 'Jane Doe', email: 'jane@example.com' }],
      };
      const updatedEvent = { ...mockEvent, ...updateData };
      mockApiClient.put.mockResolvedValueOnce(updatedEvent);

      const result = await updateEvent(1, updateData);

      expect(mockApiClient.put).toHaveBeenCalledWith('/api/events/1', updateData);
      expect(result).toEqual(updatedEvent);
    });

    it('updates event with partial fields', async () => {
      const updateData = { title: 'Only Title Updated' };
      const updatedEvent = { ...mockEvent, title: 'Only Title Updated' };
      mockApiClient.put.mockResolvedValueOnce(updatedEvent);

      const result = await updateEvent(1, updateData);

      expect(mockApiClient.put).toHaveBeenCalledWith('/api/events/1', updateData);
      expect(result.title).toBe('Only Title Updated');
    });
  });

  describe('confirmEvent', () => {
    it('confirms an event', async () => {
      const confirmedEvent = { ...mockEvent, status: 'confirmed' as const };
      mockApiClient.post.mockResolvedValueOnce(confirmedEvent);

      const result = await confirmEvent(1);

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/events/1/confirm');
      expect(result.status).toBe('confirmed');
    });
  });

  describe('rejectEvent', () => {
    it('rejects an event', async () => {
      const rejectedEvent = { ...mockEvent, status: 'rejected' as const };
      mockApiClient.post.mockResolvedValueOnce(rejectedEvent);

      const result = await rejectEvent(1);

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/events/1/reject');
      expect(result.status).toBe('rejected');
    });
  });

  describe('getChannelHistory', () => {
    it('fetches message history for a channel', async () => {
      const mockHistory: MessageHistory[] = [
        {
          id: 1,
          channel_id: 1,
          sender_jid: '1234567890@s.whatsapp.net',
          sender_name: 'John Doe',
          message_text: 'First message',
          timestamp: '2024-01-14T09:00:00Z',
        },
        {
          id: 2,
          channel_id: 1,
          sender_jid: '0987654321@s.whatsapp.net',
          sender_name: 'Jane Doe',
          message_text: 'Second message',
          timestamp: '2024-01-14T09:05:00Z',
        },
      ];
      mockApiClient.get.mockResolvedValueOnce(mockHistory);

      const result = await getChannelHistory(1);

      expect(mockApiClient.get).toHaveBeenCalledWith('/api/events/channel/1/history');
      expect(result).toEqual(mockHistory);
      expect(result).toHaveLength(2);
    });

    it('returns empty array when no history', async () => {
      mockApiClient.get.mockResolvedValueOnce([]);

      const result = await getChannelHistory(999);

      expect(result).toEqual([]);
    });
  });

  describe('listCalendars', () => {
    it('fetches available calendars', async () => {
      const mockCalendars: Calendar[] = [
        { id: 'primary', summary: 'Primary Calendar', primary: true },
        { id: 'work', summary: 'Work Calendar', primary: false },
        { id: 'personal', summary: 'Personal Calendar', primary: false },
      ];
      mockApiClient.get.mockResolvedValueOnce(mockCalendars);

      const result = await listCalendars();

      expect(mockApiClient.get).toHaveBeenCalledWith('/api/gcal/calendars');
      expect(result).toEqual(mockCalendars);
      expect(result.find((c) => c.primary)).toBeTruthy();
    });

    it('returns empty array when no calendars', async () => {
      mockApiClient.get.mockResolvedValueOnce([]);

      const result = await listCalendars();

      expect(result).toEqual([]);
    });
  });
});
