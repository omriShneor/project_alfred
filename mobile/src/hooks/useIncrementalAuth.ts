import { useMutation, useQueryClient } from '@tanstack/react-query';
import { requestAdditionalScopes, exchangeAddScopesCode, ScopeType } from '../api/auth';

export function useRequestAdditionalScopes() {
  return useMutation({
    mutationFn: ({ scopes, redirectUri }: { scopes: ScopeType[]; redirectUri?: string }) =>
      requestAdditionalScopes(scopes, redirectUri),
  });
}

export function useExchangeAddScopesCode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ code, scopes, redirectUri }: { code: string; scopes: ScopeType[]; redirectUri?: string }) =>
      exchangeAddScopesCode(code, scopes, redirectUri),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['gcalStatus'] });
      queryClient.invalidateQueries({ queryKey: ['onboardingStatus'] });
      queryClient.invalidateQueries({ queryKey: ['gmailStatus'] });
    },
  });
}
