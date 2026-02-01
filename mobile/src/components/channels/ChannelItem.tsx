import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { Card } from '../common/Card';
import { Badge } from '../common/Badge';
import { Button } from '../common/Button';
import { useCreateChannel, useDeleteChannel, useChannels } from '../../hooks/useChannels';
import { useCreateTelegramChannel, useDeleteTelegramChannel, useTelegramChannels } from '../../hooks/useTelegram';
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

  // Find the tracked channel data if this channel is tracked
  const trackedChannel = trackedChannels?.find(
    (tc) => tc.identifier === channel.identifier
  );

  const handleTrack = () => {
    if (sourceType === 'telegram') {
      // Telegram uses 'contact' type for contacts
      createTgChannel.mutate({
        type: 'contact',
        identifier: channel.identifier,
        name: channel.name,
      }, { onSuccess: () => onTrack?.() });
    } else {
      // WhatsApp uses 'sender' type for contacts
      createWaChannel.mutate({
        type: 'sender',
        identifier: channel.identifier,
        name: channel.name,
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

      <View style={styles.actions}>
        {channel.is_tracked ? (
          <Button
            title="Untrack"
            onPress={handleUntrack}
            variant="danger"
            size="small"
            loading={isLoading}
            style={styles.button}
          />
        ) : (
          <Button
            title="Track"
            onPress={handleTrack}
            variant="success"
            size="small"
            loading={isLoading}
            style={styles.button}
          />
        )}
      </View>
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
    justifyContent: 'flex-end',
  },
  button: {
    minWidth: 80,
  },
});
