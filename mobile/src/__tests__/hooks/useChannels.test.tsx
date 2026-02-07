import React from 'react';
import { renderHook, waitFor, act } from '@testing-library/react-native';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  useChannels,
  useCreateChannel,
  useUpdateChannel,
  useDeleteChannel,
  useWhatsAppTopContacts,
  useAddWhatsAppCustomSource,
} from '../../hooks/useChannels';
import * as channelsApi from '../../api/channels';
import type { Channel, SourceTopContact } from '../../types/channel';

// Mock the API module
jest.mock('../../api/channels');

const mockChannelsApi = channelsApi as jest.Mocked<typeof channelsApi>;

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

describe('useChannels hooks', () => {
  const mockChannel: Channel = {
    id: 1,
    type: 'sender',
    source_type: 'whatsapp',
    identifier: '1234567890@s.whatsapp.net',
    name: 'John Doe',
    enabled: true,
    created_at: '2024-01-14T10:00:00Z',
  };

  const mockTopContact: SourceTopContact = {
    identifier: '1234567890@s.whatsapp.net',
    name: 'John Doe',
    message_count: 50,
    is_tracked: false,
    type: 'sender',
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('useChannels', () => {
    it('fetches channels without type filter', async () => {
      mockChannelsApi.listChannels.mockResolvedValueOnce([mockChannel]);

      const { result } = renderHook(() => useChannels(), { wrapper: createWrapper() });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual([mockChannel]);
      expect(mockChannelsApi.listChannels).toHaveBeenCalledWith(undefined);
    });

    it('fetches channels with type filter', async () => {
      mockChannelsApi.listChannels.mockResolvedValueOnce([mockChannel]);

      const { result } = renderHook(() => useChannels('sender'), { wrapper: createWrapper() });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockChannelsApi.listChannels).toHaveBeenCalledWith('sender');
    });

    it('handles loading state', async () => {
      mockChannelsApi.listChannels.mockImplementationOnce(
        () => new Promise((resolve) => setTimeout(() => resolve([mockChannel]), 100))
      );

      const { result } = renderHook(() => useChannels(), { wrapper: createWrapper() });

      expect(result.current.isLoading).toBe(true);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });
    });

    it('handles error state', async () => {
      mockChannelsApi.listChannels.mockRejectedValueOnce(new Error('Failed to fetch'));

      const { result } = renderHook(() => useChannels(), { wrapper: createWrapper() });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });

      expect(result.current.error?.message).toBe('Failed to fetch');
    });

    it('returns empty array when no channels', async () => {
      mockChannelsApi.listChannels.mockResolvedValueOnce([]);

      const { result } = renderHook(() => useChannels(), { wrapper: createWrapper() });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual([]);
    });
  });

  describe('useCreateChannel', () => {
    it('creates channel successfully', async () => {
      mockChannelsApi.createChannel.mockResolvedValueOnce(mockChannel);

      const { result } = renderHook(() => useCreateChannel(), { wrapper: createWrapper() });

      await act(async () => {
        result.current.mutate({
          type: 'sender',
          identifier: '1234567890@s.whatsapp.net',
          name: 'John Doe',
        });
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockChannelsApi.createChannel).toHaveBeenCalledWith({
        type: 'sender',
        identifier: '1234567890@s.whatsapp.net',
        name: 'John Doe',
      });
    });

    it('handles create error', async () => {
      mockChannelsApi.createChannel.mockRejectedValueOnce(new Error('Create failed'));

      const { result } = renderHook(() => useCreateChannel(), { wrapper: createWrapper() });

      await act(async () => {
        result.current.mutate({
          type: 'sender',
          identifier: '1234567890@s.whatsapp.net',
          name: 'John Doe',
        });
      });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });

      expect(result.current.error?.message).toBe('Create failed');
    });
  });

  describe('useUpdateChannel', () => {
    it('updates channel successfully', async () => {
      const updatedChannel = { ...mockChannel, name: 'John Smith' };
      mockChannelsApi.updateChannel.mockResolvedValueOnce(updatedChannel);

      const { result } = renderHook(() => useUpdateChannel(), { wrapper: createWrapper() });

      await act(async () => {
        result.current.mutate({ id: 1, data: { name: 'John Smith' } });
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockChannelsApi.updateChannel).toHaveBeenCalledWith(1, { name: 'John Smith' });
    });

    it('updates channel enabled status', async () => {
      const updatedChannel = { ...mockChannel, enabled: false };
      mockChannelsApi.updateChannel.mockResolvedValueOnce(updatedChannel);

      const { result } = renderHook(() => useUpdateChannel(), { wrapper: createWrapper() });

      await act(async () => {
        result.current.mutate({ id: 1, data: { enabled: false } });
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });
    });
  });

  describe('useDeleteChannel', () => {
    it('deletes channel successfully', async () => {
      mockChannelsApi.deleteChannel.mockResolvedValueOnce(undefined);

      const { result } = renderHook(() => useDeleteChannel(), { wrapper: createWrapper() });

      await act(async () => {
        result.current.mutate(1);
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockChannelsApi.deleteChannel).toHaveBeenCalledWith(1);
    });

    it('handles delete error', async () => {
      mockChannelsApi.deleteChannel.mockRejectedValueOnce(new Error('Delete failed'));

      const { result } = renderHook(() => useDeleteChannel(), { wrapper: createWrapper() });

      await act(async () => {
        result.current.mutate(1);
      });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });
    });
  });

  describe('useWhatsAppTopContacts', () => {
    it('fetches top contacts when enabled', async () => {
      mockChannelsApi.getWhatsAppTopContacts.mockResolvedValueOnce([mockTopContact]);

      const { result } = renderHook(() => useWhatsAppTopContacts(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual([mockTopContact]);
    });

    it('does not fetch when disabled', async () => {
      const { result } = renderHook(() => useWhatsAppTopContacts({ enabled: false }), {
        wrapper: createWrapper(),
      });

      expect(result.current.fetchStatus).toBe('idle');
      expect(mockChannelsApi.getWhatsAppTopContacts).not.toHaveBeenCalled();
    });

    it('enabled by default', async () => {
      mockChannelsApi.getWhatsAppTopContacts.mockResolvedValueOnce([]);

      const { result } = renderHook(() => useWhatsAppTopContacts(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockChannelsApi.getWhatsAppTopContacts).toHaveBeenCalled();
    });
  });

  describe('useAddWhatsAppCustomSource', () => {
    it('adds custom source successfully', async () => {
      mockChannelsApi.addWhatsAppCustomSource.mockResolvedValueOnce(mockChannel);

      const { result } = renderHook(() => useAddWhatsAppCustomSource(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate('Alice Johnson');
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockChannelsApi.addWhatsAppCustomSource).toHaveBeenCalledWith('Alice Johnson');
    });

    it('handles add custom source error', async () => {
      mockChannelsApi.addWhatsAppCustomSource.mockRejectedValueOnce(
        new Error('Contact not found')
      );

      const { result } = renderHook(() => useAddWhatsAppCustomSource(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.mutate('Missing Contact');
      });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });

      expect(result.current.error?.message).toBe('Contact not found');
    });
  });
});
