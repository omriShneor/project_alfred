import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  getFeatures,
  updateSmartCalendar,
  getSmartCalendarStatus,
} from '../api/features';
import type { UpdateSmartCalendarRequest } from '../types/features';

// Get all feature settings
export function useFeatures() {
  return useQuery({
    queryKey: ['features'],
    queryFn: getFeatures,
    refetchInterval: 5000, // Poll every 5 seconds during setup
  });
}

// Update Smart Calendar settings
export function useUpdateSmartCalendar() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: UpdateSmartCalendarRequest) => updateSmartCalendar(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['features'] });
      queryClient.invalidateQueries({ queryKey: ['smartCalendarStatus'] });
    },
  });
}

// Get detailed Smart Calendar status (used during setup)
export function useSmartCalendarStatus() {
  return useQuery({
    queryKey: ['smartCalendarStatus'],
    queryFn: getSmartCalendarStatus,
    refetchInterval: 3000, // Poll frequently during permission setup
  });
}

// Helper hook to check if Smart Calendar is enabled and setup is complete
export function useSmartCalendarEnabled() {
  const { data: features } = useFeatures();

  return {
    enabled: features?.smart_calendar?.enabled ?? false,
    setupComplete: features?.smart_calendar?.setup_complete ?? false,
    isReady: (features?.smart_calendar?.enabled && features?.smart_calendar?.setup_complete) ?? false,
  };
}
