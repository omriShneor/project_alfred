import React, { useMemo } from 'react';
import { FlatList, Text, View, StyleSheet, RefreshControl } from 'react-native';
import { ChannelItem } from './ChannelItem';
import { colors } from '../../theme/colors';
import type { DiscoverableChannel } from '../../types/channel';

interface ChannelListProps {
  channels: DiscoverableChannel[];
  refreshing?: boolean;
  onRefresh?: () => void;
  onTrack?: () => void;
}

export function ChannelList({ channels, refreshing, onRefresh, onTrack }: ChannelListProps) {
  // Sort channels: tracked first, then untracked
  const sortedChannels = useMemo(() => {
    return [...channels].sort((a, b) => {
      if (a.is_tracked === b.is_tracked) return 0;
      return a.is_tracked ? -1 : 1;
    });
  }, [channels]);

  if (channels.length === 0) {
    return (
      <View style={styles.empty}>
        <Text style={styles.emptyText}>No contacts/groups found</Text>
        <Text style={styles.emptySubtext}>
          Make sure WhatsApp is connected
        </Text>
      </View>
    );
  }

  return (
    <FlatList
      data={sortedChannels}
      keyExtractor={(item) => item.identifier}
      renderItem={({ item }) => <ChannelItem channel={item} onTrack={onTrack} />}
      contentContainerStyle={styles.list}
      refreshControl={
        onRefresh ? (
          <RefreshControl refreshing={refreshing || false} onRefresh={onRefresh} />
        ) : undefined
      }
    />
  );
}

const styles = StyleSheet.create({
  list: {
    paddingBottom: 20,
  },
  empty: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 40,
  },
  emptyText: {
    fontSize: 16,
    fontWeight: '600',
    color: colors.text,
  },
  emptySubtext: {
    fontSize: 14,
    color: colors.textSecondary,
    marginTop: 8,
  },
});
