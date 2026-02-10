export type ReminderStatus = 'pending' | 'confirmed' | 'synced' | 'rejected' | 'completed' | 'dismissed';
export type ReminderPriority = 'low' | 'normal' | 'high';
export type ReminderActionType = 'create' | 'update' | 'delete';

export interface Reminder {
  id: number;
  channel_id: number;
  channel_name?: string;
  google_event_id?: string;
  calendar_id: string;
  title: string;
  description: string;
  location?: string;
  due_date?: string;
  reminder_time?: string;
  priority: ReminderPriority;
  status: ReminderStatus;
  action_type: ReminderActionType;
  original_msg_id?: number;
  llm_reasoning: string;
  llm_confidence?: number;
  quality_flags?: string[];
  source?: string;
  email_source_id?: number;
  created_at: string;
  updated_at: string;
}

export interface ReminderWithMessage extends Reminder {
  trigger_message?: {
    id: number;
    channel_id: number;
    sender_jid: string;
    sender_name: string;
    message_text: string;
    timestamp: string;
  };
}

export interface UpdateReminderRequest {
  title?: string;
  description?: string;
  location?: string;
  due_date?: string;
  reminder_time?: string;
  priority?: ReminderPriority;
}

export interface CreateReminderRequest {
  title: string;
  description?: string;
  location?: string;
  due_date?: string;
  reminder_time?: string;
  priority?: ReminderPriority;
}
