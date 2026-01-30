import React, { useState, useMemo } from 'react';
import { View, StyleSheet } from 'react-native';
import { SearchInput, FilterChips, LoadingSpinner } from '../../components/common';
import { ChannelList, ChannelStats } from '../../components/channels';
import { colors } from '../../theme/colors';
import { useDiscoverableTelegramChannels, useDebounce, useTelegramStatus } from '../../hooks';

const TYPE_FILTERS = [
  { label: 'All', value: 'all' },
  { label: 'Contacts', value: 'contact' },
  { label: 'Groups', value: 'group' },
  { label: 'Channels', value: 'channel' },
];

export function TelegramPreferencesScreen() {
  const { data: telegramStatus } = useTelegramStatus();

  const [searchQuery, setSearchQuery] = useState('');
  const [typeFilter, setTypeFilter] = useState('all');
  const debouncedSearch = useDebounce(searchQuery, 150);

  const {
    data: channels,
    isLoading: channelsLoading,
    refetch: refetchChannels,
    isRefetching: isRefetchingChannels,
  } = useDiscoverableTelegramChannels();

  const filteredChannels = useMemo(() => {
    if (!channels) return [];

    return channels.filter((channel) => {
      if (typeFilter !== 'all' && channel.type !== typeFilter) {
        return false;
      }
      if (debouncedSearch) {
        const search = debouncedSearch.toLowerCase();
        return channel.name.toLowerCase().includes(search);
      }
      return true;
    });
  }, [channels, typeFilter, debouncedSearch]);

  return (
    <View style={styles.container}>
      <SearchInput
        value={searchQuery}
        onChangeText={setSearchQuery}
        placeholder="Search contacts/groups/channels..."
      />
      <FilterChips
        options={TYPE_FILTERS}
        selected={typeFilter}
        onSelect={setTypeFilter}
      />
      {channels && <ChannelStats channels={channels} />}
      {channelsLoading ? (
        <LoadingSpinner message="Loading contacts/groups..." />
      ) : (
        <ChannelList
          channels={filteredChannels}
          refreshing={isRefetchingChannels}
          onRefresh={refetchChannels}
          onTrack={() => setSearchQuery('')}
          sourceType="telegram"
        />
      )}
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
