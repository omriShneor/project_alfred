export interface TodayEvent {
  id: string;
  summary: string;
  description?: string;
  location?: string;
  start_time: string;
  end_time: string;
  all_day: boolean;
  calendar_id: string;
  source?: 'alfred' | 'google' | 'outlook'; // Which calendar this event came from
}
