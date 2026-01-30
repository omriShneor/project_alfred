import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  getTelegramStatus,
  sendTelegramCode,
  verifyTelegramCode,
  disconnectTelegram,
  reconnectTelegram,
  discoverTelegramChannels,
  listTelegramChannels,
  createTelegramChannel,
  updateTelegramChannel,
  deleteTelegramChannel,
  type TelegramStatus,
  type CreateTelegramChannelRequest,
  type UpdateTelegramChannelRequest,
} from '../api/telegram';

// Query key constants
const TELEGRAM_STATUS_KEY = ['telegramStatus'];
const TELEGRAM_CHANNELS_KEY = ['telegramChannels'];
const TELEGRAM_DISCOVERABLE_KEY = ['telegramDiscoverable'];

// Hook to get Telegram status
export function useTelegramStatus() {
  return useQuery<TelegramStatus>({
    queryKey: TELEGRAM_STATUS_KEY,
    queryFn: getTelegramStatus,
    refetchInterval: 5000, // Poll every 5 seconds during connection
  });
}

// Hook to send verification code
export function useSendTelegramCode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (phoneNumber: string) => sendTelegramCode(phoneNumber),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: TELEGRAM_STATUS_KEY });
    },
  });
}

// Hook to verify code
export function useVerifyTelegramCode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (code: string) => verifyTelegramCode(code),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: TELEGRAM_STATUS_KEY });
      queryClient.invalidateQueries({ queryKey: ['appStatus'] });
    },
  });
}

// Hook to disconnect Telegram
export function useDisconnectTelegram() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: disconnectTelegram,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: TELEGRAM_STATUS_KEY });
      queryClient.invalidateQueries({ queryKey: ['appStatus'] });
    },
  });
}

// Hook to reconnect Telegram
export function useReconnectTelegram() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: reconnectTelegram,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: TELEGRAM_STATUS_KEY });
      queryClient.invalidateQueries({ queryKey: ['appStatus'] });
    },
  });
}

// Hook to get discoverable Telegram channels
export function useDiscoverableTelegramChannels() {
  return useQuery({
    queryKey: TELEGRAM_DISCOVERABLE_KEY,
    queryFn: discoverTelegramChannels,
  });
}

// Hook to get tracked Telegram channels
export function useTelegramChannels() {
  return useQuery({
    queryKey: TELEGRAM_CHANNELS_KEY,
    queryFn: listTelegramChannels,
  });
}

// Hook to create a Telegram channel
export function useCreateTelegramChannel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateTelegramChannelRequest) => createTelegramChannel(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: TELEGRAM_CHANNELS_KEY });
      queryClient.invalidateQueries({ queryKey: TELEGRAM_DISCOVERABLE_KEY });
    },
  });
}

// Hook to update a Telegram channel
export function useUpdateTelegramChannel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateTelegramChannelRequest }) =>
      updateTelegramChannel(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: TELEGRAM_CHANNELS_KEY });
    },
  });
}

// Hook to delete a Telegram channel
export function useDeleteTelegramChannel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => deleteTelegramChannel(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: TELEGRAM_CHANNELS_KEY });
      queryClient.invalidateQueries({ queryKey: TELEGRAM_DISCOVERABLE_KEY });
    },
  });
}
