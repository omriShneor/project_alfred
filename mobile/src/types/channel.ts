export type ChannelType = 'sender' | 'group' | 'channel';
export type SourceType = 'whatsapp' | 'telegram' | 'gmail';

export interface Channel {
  id: number;
  source_type?: SourceType;
  type: ChannelType;
  identifier: string;
  name: string;
  calendar_id: string;
  enabled: boolean;
  created_at: string;
}

export interface DiscoverableChannel {
  type: ChannelType | 'contact' | 'channel'; // Telegram uses different type names
  identifier: string;
  name: string;
  is_tracked: boolean;
  channel_id?: number;
}

export interface CreateChannelRequest {
  type: ChannelType;
  identifier: string;
  name: string;
  calendar_id: string;
}

export interface UpdateChannelRequest {
  name?: string;
  calendar_id?: string;
  enabled?: boolean;
}
