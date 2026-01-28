import React, { useState, useMemo } from 'react';
import { View, StyleSheet } from 'react-native';
import { SearchInput, FilterChips, LoadingSpinner } from '../../components/common';
import { ChannelList, ChannelStats } from '../../components/channels';
import { colors } from '../../theme/colors';
import { useDiscoverableChannels, useDebounce, useWhatsAppStatus } from '../../hooks';

const TYPE_FILTERS = [
  { label: 'All', value: 'all' },
  { label: 'Contacts', value: 'sender' },
  { label: 'Groups', value: 'group' },
];

export function WhatsAppPreferencesScreen() {
  const { data: waStatus } = useWhatsAppStatus();

  const [searchQuery, setSearchQuery] = useState('');
  const [typeFilter, setTypeFilter] = useState('all');
  const debouncedSearch = useDebounce(searchQuery, 150);

  const {
    data: channels,
    isLoading: channelsLoading,
    refetch: refetchChannels,
    isRefetching: isRefetchingChannels,
  } = useDiscoverableChannels({ enabled: waStatus?.connected ?? false });

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
        placeholder="Search contacts/groups..."
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
