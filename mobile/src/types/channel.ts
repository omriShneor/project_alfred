export type ChannelType = 'sender' | 'contact'; // contacts only (sender for WhatsApp, contact for Telegram)
export type SourceType = 'whatsapp' | 'telegram' | 'gmail';

export interface Channel {
  id: number;
  source_type?: SourceType;
  type: ChannelType;
  identifier: string;
  name: string;
  enabled: boolean;
  created_at: string;
}

export interface DiscoverableChannel {
  type: ChannelType; // 'sender' for WhatsApp, 'contact' for Telegram
  identifier: string;
  name: string;
  is_tracked: boolean;
  channel_id?: number;
}

export interface CreateChannelRequest {
  type: ChannelType;
  identifier: string;
  name: string;
}

export interface UpdateChannelRequest {
  name?: string;
  enabled?: boolean;
}

// Top Contact for Add Source modal (WhatsApp/Telegram)
export interface SourceTopContact {
  identifier: string;
  name: string;
  message_count: number;
  is_tracked: boolean;
  channel_id?: number;
  type: ChannelType; // 'sender' for WhatsApp, 'contact' for Telegram
}

// Request to add a custom source (phone number for WhatsApp, username for Telegram)
export interface AddCustomSourceRequest {
  value: string; // phone_number for WhatsApp, username for Telegram
}
