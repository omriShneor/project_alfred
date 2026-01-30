import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getAppStatus, completeOnboarding } from '../api/app';
import type { CompleteOnboardingRequest } from '../types/app';

export function useAppStatus() {
  return useQuery({
    queryKey: ['appStatus'],
    queryFn: getAppStatus,
  });
}

export function useCompleteOnboarding() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CompleteOnboardingRequest) => completeOnboarding(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['appStatus'] });
    },
  });
}

export function useIsOnboarded() {
  const { data: appStatus, isLoading } = useAppStatus();

  return {
    isOnboarded: appStatus?.onboarding_complete ?? false,
    isLoading,
  };
}
