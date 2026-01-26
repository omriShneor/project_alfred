import { useQuery } from '@tanstack/react-query';
import { getTodayEvents } from '../api/calendar';
import type { TodayEvent } from '../types/calendar';

export function useTodayEvents(calendarId?: string) {
  return useQuery<TodayEvent[]>({
    queryKey: ['todayEvents', calendarId],
    queryFn: () => getTodayEvents(calendarId),
    refetchInterval: 60000, // Refresh every minute
  });
}
