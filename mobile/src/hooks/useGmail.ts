import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  getGmailStatus,
  getGmailSettings,
  updateGmailSettings,
  discoverCategories,
  discoverSenders,
  discoverDomains,
  listEmailSources,
  createEmailSource,
  updateEmailSource,
  deleteEmailSource,
} from '../api/gmail';
import type {
  GmailSettings,
  EmailSourceType,
  CreateEmailSourceRequest,
  UpdateEmailSourceRequest,
} from '../types/gmail';

export function useGmailStatus() {
  return useQuery({
    queryKey: ['gmailStatus'],
    queryFn: getGmailStatus,
    refetchInterval: 10000, // Poll every 10 seconds
  });
}

export function useGmailSettings() {
  return useQuery({
    queryKey: ['gmailSettings'],
    queryFn: getGmailSettings,
  });
}

export function useUpdateGmailSettings() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (settings: Partial<GmailSettings>) => updateGmailSettings(settings),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['gmailSettings'] });
      queryClient.invalidateQueries({ queryKey: ['gmailStatus'] });
    },
  });
}

export function useDiscoverCategories() {
  return useQuery({
    queryKey: ['gmailDiscoverCategories'],
    queryFn: discoverCategories,
    enabled: false, // Only fetch when explicitly triggered
  });
}

export function useDiscoverSenders(limit?: number) {
  return useQuery({
    queryKey: ['gmailDiscoverSenders', limit],
    queryFn: () => discoverSenders(limit),
    enabled: false, // Only fetch when explicitly triggered
  });
}

export function useDiscoverDomains(limit?: number) {
  return useQuery({
    queryKey: ['gmailDiscoverDomains', limit],
    queryFn: () => discoverDomains(limit),
    enabled: false, // Only fetch when explicitly triggered
  });
}

export function useEmailSources(type?: EmailSourceType) {
  return useQuery({
    queryKey: ['emailSources', type],
    queryFn: () => listEmailSources(type),
  });
}

export function useCreateEmailSource() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateEmailSourceRequest) => createEmailSource(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['emailSources'] });
    },
  });
}

export function useUpdateEmailSource() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateEmailSourceRequest }) =>
      updateEmailSource(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['emailSources'] });
    },
  });
}

export function useDeleteEmailSource() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => deleteEmailSource(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['emailSources'] });
    },
  });
}
