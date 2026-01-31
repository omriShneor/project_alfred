import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  listChannels,
  createChannel,
  updateChannel,
  deleteChannel,
  discoverChannels,
  getWhatsAppTopContacts,
  addWhatsAppCustomSource,
} from '../api/channels';
import type {
  Channel,
  DiscoverableChannel,
  CreateChannelRequest,
  UpdateChannelRequest,
  SourceTopContact,
} from '../types/channel';

export function useChannels(type?: string) {
  return useQuery<Channel[]>({
    queryKey: ['channels', type],
    queryFn: () => listChannels(type),
  });
}

export function useDiscoverableChannels(options?: { enabled?: boolean }) {
  return useQuery<DiscoverableChannel[]>({
    queryKey: ['discoverableChannels'],
    queryFn: discoverChannels,
    enabled: options?.enabled ?? true,
  });
}

export function useCreateChannel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateChannelRequest) => createChannel(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['channels'] });
      queryClient.invalidateQueries({ queryKey: ['discoverableChannels'] });
    },
  });
}

export function useUpdateChannel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateChannelRequest }) =>
      updateChannel(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['channels'] });
    },
  });
}

export function useDeleteChannel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => deleteChannel(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['channels'] });
      queryClient.invalidateQueries({ queryKey: ['discoverableChannels'] });
    },
  });
}

// WhatsApp top contacts hook
export function useWhatsAppTopContacts(options?: { enabled?: boolean }) {
  return useQuery<SourceTopContact[]>({
    queryKey: ['whatsappTopContacts'],
    queryFn: getWhatsAppTopContacts,
    enabled: options?.enabled ?? true,
  });
}

// WhatsApp custom source mutation
export function useAddWhatsAppCustomSource() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ phoneNumber, calendarId }: { phoneNumber: string; calendarId: string }) =>
      addWhatsAppCustomSource(phoneNumber, calendarId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['channels'] });
      queryClient.invalidateQueries({ queryKey: ['whatsappTopContacts'] });
      queryClient.invalidateQueries({ queryKey: ['discoverableChannels'] });
    },
  });
}
