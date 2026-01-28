export interface GmailStatus {
  connected: boolean;
  enabled: boolean;
  message: string;
  has_scopes: boolean;
  poll_interval_minutes: number;
  last_poll_at?: string;
}

export interface GmailSettings {
  enabled: boolean;
  poll_interval_minutes: number;
  last_poll_at?: string;
}

export type EmailSourceType = 'category' | 'sender' | 'domain';

export interface EmailSource {
  id: number;
  type: EmailSourceType;
  identifier: string;
  name: string;
  enabled: boolean;
  calendar_id: string;
  created_at: string;
  updated_at: string;
}

export interface CreateEmailSourceRequest {
  type: EmailSourceType;
  identifier: string;
  name: string;
  calendar_id: string;
}

export interface UpdateEmailSourceRequest {
  enabled?: boolean;
  calendar_id?: string;
}

export interface DiscoveredCategory {
  id: string;
  name: string;
  description: string;
  email_count: number;
}

export interface DiscoveredSender {
  email: string;
  name: string;
  email_count: number;
}

export interface DiscoveredDomain {
  domain: string;
  email_count: number;
}
