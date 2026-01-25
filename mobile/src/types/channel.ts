export type ChannelType = 'sender' | 'group';

export interface Channel {
  id: number;
  type: ChannelType;
  identifier: string;
  name: string;
  calendar_id: string;
  enabled: boolean;
  created_at: string;
}

export interface DiscoverableChannel {
  type: ChannelType;
  identifier: string;
  name: string;
  is_tracked: boolean;
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
