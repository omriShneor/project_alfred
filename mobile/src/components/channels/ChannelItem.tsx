import React, { useState } from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { Card } from '../common/Card';
import { Badge } from '../common/Badge';
import { Button } from '../common/Button';
import { CalendarPicker } from './CalendarPicker';
import { useCreateChannel, useDeleteChannel, useChannels } from '../../hooks/useChannels';
import { useCreateTelegramChannel, useDeleteTelegramChannel, useTelegramChannels } from '../../hooks/useTelegram';
import { useCalendars } from '../../hooks/useEvents';
import { useGCalStatus } from '../../hooks';
import { colors } from '../../theme/colors';
import type { DiscoverableChannel, SourceType } from '../../types/channel';

interface ChannelItemProps {
  channel: DiscoverableChannel;
  onTrack?: () => void;
  sourceType?: SourceType;
}

export function ChannelItem({ channel, onTrack, sourceType = 'whatsapp' }: ChannelItemProps) {
  // WhatsApp hooks
  const { data: waTrackedChannels } = useChannels();
  const createWaChannel = useCreateChannel();
  const deleteWaChannel = useDeleteChannel();

  // Telegram hooks
  const { data: tgTrackedChannels } = useTelegramChannels();
  const createTgChannel = useCreateTelegramChannel();
  const deleteTgChannel = useDeleteTelegramChannel();

  // Select the appropriate data based on sourceType
  const trackedChannels = sourceType === 'telegram' ? tgTrackedChannels : waTrackedChannels;

  const { data: gcalStatus } = useGCalStatus();
  const googleConnected = gcalStatus?.connected ?? false;
  const { data: calendars } = useCalendars(googleConnected);

  // Find the tracked channel data if this channel is tracked
  const trackedChannel = trackedChannels?.find(
    (tc) => tc.identifier === channel.identifier
  );

  const [selectedCalendar, setSelectedCalendar] = useState(
    trackedChannel?.calendar_id || ''
  );

  const handleTrack = () => {
    // Find primary calendar as default if not selected
    const calendarId = selectedCalendar || calendars?.find((c) => c.primary)?.id || calendars?.[0]?.id || '';

    if (!calendarId) {
      return; // No calendar available
    }

    if (sourceType === 'telegram') {
      // Telegram uses 'contact' type for contacts
      createTgChannel.mutate({
        type: 'contact',
        identifier: channel.identifier,
        name: channel.name,
        calendar_id: calendarId,
      }, { onSuccess: () => onTrack?.() });
    } else {
      // WhatsApp uses 'sender' type for contacts
      createWaChannel.mutate({
        type: 'sender',
        identifier: channel.identifier,
        name: channel.name,
        calendar_id: calendarId,
      }, { onSuccess: () => onTrack?.() });
    }
  };

  const handleUntrack = () => {
    if (trackedChannel) {
      if (sourceType === 'telegram') {
        deleteTgChannel.mutate(trackedChannel.id);
      } else {
        deleteWaChannel.mutate(trackedChannel.id);
      }
    }
  };

  const isLoading = sourceType === 'telegram'
    ? createTgChannel.isPending || deleteTgChannel.isPending
    : createWaChannel.isPending || deleteWaChannel.isPending;

  return (
    <Card>
      <View style={styles.header}>
        <View style={styles.headerLeft}>
          <Badge
            label="Contact"
            variant="sender"
          />
          <Text style={styles.name} numberOfLines={1}>
            {channel.name}
          </Text>
        </View>
        <Badge
          label={channel.is_tracked ? 'Tracked' : 'Not tracked'}
          bgColor={channel.is_tracked ? colors.success + '20' : colors.border}
          textColor={channel.is_tracked ? colors.success : colors.textSecondary}
        />
      </View>

      {channel.is_tracked ? (
        <View style={styles.actions}>
          <View style={styles.calendarRow}>
            <CalendarPicker
              value={trackedChannel?.calendar_id || selectedCalendar}
              onChange={setSelectedCalendar}
              enabled={googleConnected}
            />
          </View>
          <Button
            title="Untrack"
            onPress={handleUntrack}
            variant="danger"
            size="small"
            loading={isLoading}
            style={styles.button}
          />
        </View>
      ) : (
        <View style={styles.actions}>
          <View style={styles.calendarRow}>
            <CalendarPicker
              value={selectedCalendar}
              onChange={setSelectedCalendar}
              enabled={googleConnected}
            />
          </View>
          <Button
            title="Track"
            onPress={handleTrack}
            variant="success"
            size="small"
            loading={isLoading}
            style={styles.button}
          />
        </View>
      )}
    </Card>
  );
}

const styles = StyleSheet.create({
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 12,
  },
  headerLeft: {
    flexDirection: 'row',
    alignItems: 'center',
    flex: 1,
    marginRight: 8,
  },
  name: {
    fontSize: 15,
    fontWeight: '600',
    color: colors.text,
    marginLeft: 8,
    flex: 1,
  },
  actions: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
  },
  calendarRow: {
    flex: 1,
  },
  button: {
    minWidth: 80,
  },
});
