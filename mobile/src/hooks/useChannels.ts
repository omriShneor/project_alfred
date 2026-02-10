import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  listChannels,
  createChannel,
  updateChannel,
  deleteChannel,
  getWhatsAppTopContacts,
  addWhatsAppCustomSource,
  searchWhatsAppContacts,
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
    refetchOnMount: 'always',
    // Keep polling while the modal is open so early partial suggestions can
    // converge to the finalized ranking after HistorySync phase 2 completes.
    refetchInterval: (options?.enabled ?? true) ? 2000 : false,
  });
}

// WhatsApp contact search hook
export function useSearchWhatsAppContacts(query: string) {
  return useQuery<SourceTopContact[]>({
    queryKey: ['whatsappContactSearch', query],
    queryFn: () => searchWhatsAppContacts(query),
    enabled: query.length >= 2,
  });
}

// WhatsApp custom source mutation
export function useAddWhatsAppCustomSource() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (contactName: string) => addWhatsAppCustomSource(contactName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['channels'] });
      queryClient.invalidateQueries({ queryKey: ['whatsappTopContacts'] });
    },
  });
}
