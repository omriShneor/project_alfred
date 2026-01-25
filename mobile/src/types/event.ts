export type EventStatus = 'pending' | 'confirmed' | 'synced' | 'rejected' | 'deleted';
export type EventActionType = 'create' | 'update' | 'delete';

export interface Attendee {
  id: number;
  event_id: number;
  name: string;
  email?: string;
}

export interface CalendarEvent {
  id: number;
  channel_id: number;
  channel_name?: string;
  google_event_id?: string;
  calendar_id: string;
  title: string;
  description: string;
  start_time: string;
  end_time?: string;
  location: string;
  status: EventStatus;
  action_type: EventActionType;
  original_msg_id?: number;
  llm_reasoning: string;
  attendees: Attendee[];
  created_at: string;
  updated_at: string;
}

export interface EventWithMessage extends CalendarEvent {
  trigger_message?: {
    id: number;
    channel_id: number;
    sender_jid: string;
    sender_name: string;
    message_text: string;
    timestamp: string;
  };
}

export interface UpdateEventRequest {
  title?: string;
  description?: string;
  start_time?: string;
  end_time?: string;
  location?: string;
  attendees?: { name: string; email?: string }[];
}

export interface MessageHistory {
  id: number;
  channel_id: number;
  sender_jid: string;
  sender_name: string;
  message_text: string;
  timestamp: string;
}

export interface Calendar {
  id: string;
  summary: string;
  primary: boolean;
}
