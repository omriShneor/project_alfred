import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  getGmailStatus,
  listEmailSources,
  createEmailSource,
  updateEmailSource,
  deleteEmailSource,
  getTopContacts,
  addCustomSource,
  searchGmailContacts,
} from '../api/gmail';
import type {
  EmailSourceType,
  CreateEmailSourceRequest,
  UpdateEmailSourceRequest,
  AddCustomSourceRequest,
  TopContact,
} from '../types/gmail';

export function useGmailStatus() {
  return useQuery({
    queryKey: ['gmailStatus'],
    queryFn: getGmailStatus,
    refetchInterval: 10000, // Poll every 10 seconds
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

// Top Contacts - cached contacts for fast discovery
export function useTopContacts() {
  return useQuery({
    queryKey: ['gmailTopContacts'],
    queryFn: getTopContacts,
  });
}

// Search all cached contacts by name or email
export function useSearchGmailContacts(query: string) {
  return useQuery<TopContact[]>({
    queryKey: ['gmailContactSearch', query],
    queryFn: () => searchGmailContacts(query),
    enabled: query.length >= 2,
  });
}

// Add custom email source
export function useAddCustomSource() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: AddCustomSourceRequest) => addCustomSource(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['emailSources'] });
      queryClient.invalidateQueries({ queryKey: ['gmailTopContacts'] });
    },
  });
}
