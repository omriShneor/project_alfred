import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  listChannels,
  createChannel,
  updateChannel,
  deleteChannel,
  discoverChannels,
} from '../api/channels';
import type {
  Channel,
  DiscoverableChannel,
  CreateChannelRequest,
  UpdateChannelRequest,
} from '../types/channel';

export function useChannels(type?: string) {
  return useQuery<Channel[]>({
    queryKey: ['channels', type],
    queryFn: () => listChannels(type),
  });
}

export function useDiscoverableChannels() {
  return useQuery<DiscoverableChannel[]>({
    queryKey: ['discoverableChannels'],
    queryFn: discoverChannels,
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
