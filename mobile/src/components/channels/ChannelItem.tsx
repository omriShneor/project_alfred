import React, { useState } from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { Card } from '../common/Card';
import { Badge } from '../common/Badge';
import { Button } from '../common/Button';
import { CalendarPicker } from './CalendarPicker';
import { useCreateChannel, useDeleteChannel, useChannels } from '../../hooks/useChannels';
import { useCalendars } from '../../hooks/useEvents';
import { colors } from '../../theme/colors';
import type { DiscoverableChannel, Channel } from '../../types/channel';

interface ChannelItemProps {
  channel: DiscoverableChannel;
}

export function ChannelItem({ channel }: ChannelItemProps) {
  const { data: trackedChannels } = useChannels();
  const { data: calendars } = useCalendars();
  const createChannel = useCreateChannel();
  const deleteChannel = useDeleteChannel();

  // Find the tracked channel data if this channel is tracked
  const trackedChannel = trackedChannels?.find(
    (tc) => tc.identifier === channel.identifier
  );

  const [selectedCalendar, setSelectedCalendar] = useState(
    trackedChannel?.calendar_id || ''
  );

  const handleTrack = () => {
    if (!selectedCalendar) {
      // Find primary calendar as default
      const primaryCal = calendars?.find((c) => c.primary);
      const calendarId = primaryCal?.id || calendars?.[0]?.id || '';

      if (!calendarId) {
        return; // No calendar available
      }

      createChannel.mutate({
        type: channel.type,
        identifier: channel.identifier,
        name: channel.name,
        calendar_id: calendarId,
      });
    } else {
      createChannel.mutate({
        type: channel.type,
        identifier: channel.identifier,
        name: channel.name,
        calendar_id: selectedCalendar,
      });
    }
  };

  const handleUntrack = () => {
    if (trackedChannel) {
      deleteChannel.mutate(trackedChannel.id);
    }
  };

  const isLoading = createChannel.isPending || deleteChannel.isPending;

  return (
    <Card>
      <View style={styles.header}>
        <View style={styles.headerLeft}>
          <Badge
            label={channel.type === 'sender' ? 'Contact' : 'Group'}
            variant={channel.type}
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
