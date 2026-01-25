import { useQuery } from '@tanstack/react-query';
import { getHealth, type HealthStatus } from '../api/health';

export function useHealth() {
  return useQuery<HealthStatus>({
    queryKey: ['health'],
    queryFn: getHealth,
    refetchInterval: 30000, // Refresh every 30 seconds
    retry: 1,
    staleTime: 10000,
  });
}
