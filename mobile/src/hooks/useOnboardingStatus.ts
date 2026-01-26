import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  getOnboardingStatus,
  getWhatsAppStatus,
  generatePairingCode,
  disconnectWhatsApp,
  getGCalStatus,
  getOAuthURL,
  exchangeOAuthCode,
} from '../api';

export function useOnboardingStatus() {
  return useQuery({
    queryKey: ['onboardingStatus'],
    queryFn: getOnboardingStatus,
    refetchInterval: 3000, // Poll every 3 seconds during onboarding
  });
}

export function useWhatsAppStatus() {
  return useQuery({
    queryKey: ['whatsappStatus'],
    queryFn: getWhatsAppStatus,
    refetchInterval: 5000,
  });
}

export function useGeneratePairingCode() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (phoneNumber: string) => generatePairingCode(phoneNumber),
    onSuccess: () => {
      // Invalidate status to trigger re-fetch
      queryClient.invalidateQueries({ queryKey: ['whatsappStatus'] });
      queryClient.invalidateQueries({ queryKey: ['onboardingStatus'] });
    },
  });
}

export function useDisconnectWhatsApp() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: disconnectWhatsApp,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['whatsappStatus'] });
      queryClient.invalidateQueries({ queryKey: ['onboardingStatus'] });
    },
  });
}

export function useGCalStatus() {
  return useQuery({
    queryKey: ['gcalStatus'],
    queryFn: getGCalStatus,
    refetchInterval: 5000,
  });
}

export function useGetOAuthURL() {
  return useMutation({
    mutationFn: (redirectUri: string) => getOAuthURL(redirectUri),
  });
}

export function useExchangeOAuthCode() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ code, redirectUri }: { code: string; redirectUri: string }) =>
      exchangeOAuthCode(code, redirectUri),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['gcalStatus'] });
      queryClient.invalidateQueries({ queryKey: ['onboardingStatus'] });
    },
  });
}
