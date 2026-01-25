import React, { useState, useMemo } from 'react';
import { View, StyleSheet } from 'react-native';
import { SearchInput, FilterChips, LoadingSpinner } from '../components/common';
import { ChannelList, ChannelStats } from '../components/channels';
import { useDiscoverableChannels } from '../hooks/useChannels';
import { useDebounce } from '../hooks/useDebounce';
import { colors } from '../theme/colors';

const TYPE_FILTERS = [
  { label: 'All', value: 'all' },
  { label: 'Contacts', value: 'sender' },
  { label: 'Groups', value: 'group' },
];

export function ChannelsScreen() {
  const [searchQuery, setSearchQuery] = useState('');
  const [typeFilter, setTypeFilter] = useState('all');

  const debouncedSearch = useDebounce(searchQuery, 150);

  const {
    data: channels,
    isLoading,
    refetch,
    isRefetching,
  } = useDiscoverableChannels();

  const filteredChannels = useMemo(() => {
    if (!channels) return [];

    return channels.filter((channel) => {
      // Type filter
      if (typeFilter !== 'all' && channel.type !== typeFilter) {
        return false;
      }

      // Search filter
      if (debouncedSearch) {
        const search = debouncedSearch.toLowerCase();
        return channel.name.toLowerCase().includes(search);
      }

      return true;
    });
  }, [channels, typeFilter, debouncedSearch]);

  if (isLoading) {
    return <LoadingSpinner message="Loading channels..." />;
  }

  return (
    <View style={styles.container}>
      <SearchInput
        value={searchQuery}
        onChangeText={setSearchQuery}
        placeholder="Search channels..."
      />

      <FilterChips
        options={TYPE_FILTERS}
        selected={typeFilter}
        onSelect={setTypeFilter}
      />

      {channels && <ChannelStats channels={channels} />}

      <ChannelList
        channels={filteredChannels}
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
});
