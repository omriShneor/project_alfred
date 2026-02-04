import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  listChannels,
  createChannel,
  updateChannel,
  deleteChannel,
  getWhatsAppTopContacts,
  addWhatsAppCustomSource,
} from '../api/channels';
import type {
  Channel,
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

export function useCreateChannel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateChannelRequest) => createChannel(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['channels'] });
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
    },
  });
}

// WhatsApp top contacts hook
export function useWhatsAppTopContacts(options?: { enabled?: boolean }) {
  return useQuery<SourceTopContact[]>({
    queryKey: ['whatsappTopContacts'],
    queryFn: getWhatsAppTopContacts,
    enabled: options?.enabled ?? true,
    staleTime: 0, // Always fetch fresh data when modal opens
  });
}

// WhatsApp custom source mutation
export function useAddWhatsAppCustomSource() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (phoneNumber: string) => addWhatsAppCustomSource(phoneNumber),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['channels'] });
      queryClient.invalidateQueries({ queryKey: ['whatsappTopContacts'] });
    },
  });
}
