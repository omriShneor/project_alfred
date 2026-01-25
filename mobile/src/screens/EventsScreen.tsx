import React, { useState, useMemo } from 'react';
import { View, StyleSheet } from 'react-native';
import { Select, LoadingSpinner } from '../components/common';
import { EventList, EventStats } from '../components/events';
import { useEvents } from '../hooks/useEvents';
import { useChannels } from '../hooks/useChannels';
import { colors } from '../theme/colors';

const STATUS_OPTIONS = [
  { label: 'All Statuses', value: '' },
  { label: 'Pending', value: 'pending' },
  { label: 'Synced', value: 'synced' },
  { label: 'Rejected', value: 'rejected' },
];

export function EventsScreen() {
  const [statusFilter, setStatusFilter] = useState('pending'); // Default to pending
  const [channelFilter, setChannelFilter] = useState('');

  const { data: channels } = useChannels();

  const channelOptions = useMemo(() => {
    const options = [{ label: 'All Channels', value: '' }];
    if (channels) {
      channels.forEach((channel) => {
        options.push({
          label: channel.name,
          value: channel.id.toString(),
        });
      });
    }
    return options;
  }, [channels]);

  const {
    data: events,
    isLoading,
    refetch,
    isRefetching,
  } = useEvents({
    status: statusFilter || undefined,
    channel_id: channelFilter ? parseInt(channelFilter) : undefined,
  });

  // Get all events for stats (unfiltered)
  const { data: allEvents } = useEvents();

  if (isLoading) {
    return <LoadingSpinner message="Loading events..." />;
  }

  return (
    <View style={styles.container}>
      <View style={styles.filters}>
        <View style={styles.filterItem}>
          <Select
            options={STATUS_OPTIONS}
            value={statusFilter}
            onChange={setStatusFilter}
            placeholder="Status"
          />
        </View>
        <View style={styles.filterItem}>
          <Select
            options={channelOptions}
            value={channelFilter}
            onChange={setChannelFilter}
            placeholder="Channel"
          />
        </View>
      </View>

      {allEvents && <EventStats events={allEvents} />}

      <EventList
        events={events || []}
        refreshing={isRefetching}
        onRefresh={refetch}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.background,
    padding: 16,
  },
  filters: {
    flexDirection: 'row',
    gap: 12,
    marginBottom: 12,
  },
  filterItem: {
    flex: 1,
  },
});
