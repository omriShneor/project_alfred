import React, { useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TouchableOpacity,
  FlatList,
  Switch,
  Alert,
} from 'react-native';
import { Feather } from '@expo/vector-icons';
import { LoadingSpinner, Card } from '../../components/common';
import { AddSourceModal } from '../../components/sources';
import { colors } from '../../theme/colors';
import {
  useChannels,
  useUpdateChannel,
  useDeleteChannel,
  useCreateChannel,
  useWhatsAppTopContacts,
  useAddWhatsAppCustomSource,
  useWhatsAppStatus,
  useSearchWhatsAppContacts,
  useDebounce,
} from '../../hooks';
import type { Channel, SourceTopContact, ChannelType } from '../../types/channel';

export function WhatsAppPreferencesScreen() {
  const { data: waStatus } = useWhatsAppStatus();
  const isConnected = waStatus?.connected ?? false;

  const [addSourceModalVisible, setAddSourceModalVisible] = useState(false);

  const { data: channels, isLoading: channelsLoading } = useChannels();
  const { data: topContacts, isLoading: contactsLoading } = useWhatsAppTopContacts({
    enabled: isConnected && addSourceModalVisible,
  });

  const [searchQuery, setSearchQuery] = useState('');
  const debouncedQuery = useDebounce(searchQuery, 300);
  const { data: searchResults, isLoading: searchLoading } = useSearchWhatsAppContacts(debouncedQuery);

  const updateChannel = useUpdateChannel();
  const deleteChannel = useDeleteChannel();
  const createChannel = useCreateChannel();
  const addCustomSource = useAddWhatsAppCustomSource();

  const handleToggleChannel = async (channel: Channel) => {
    try {
      await updateChannel.mutateAsync({
        id: channel.id,
        data: { enabled: !channel.enabled },
      });
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to update chat');
    }
  };

  const handleDeleteWhatsAppChannel = (channel: Channel) => {
    Alert.alert(
      'Remove Chat',
      `Are you sure you want to delete "${channel.name}"?`,
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Delete',
          style: 'destructive',
          onPress: async () => {
            try {
              await deleteChannel.mutateAsync(channel.id);
            } catch (error: any) {
              Alert.alert('Error', error.message || 'Failed to remove chat');
            }
          },
        },
      ]
    );
  };

  const handleOpenAddSourceModal = () => {
    if (!isConnected) {
      Alert.alert(
        'WhatsApp Not Connected',
        'Please connect WhatsApp first to add chats.',
        [{ text: 'OK' }]
      );
      return;
    }
    setAddSourceModalVisible(true);
  };

  const validateChatName = (value: string): string | null => {
    if (!value.trim()) return null;
    return null;
  };

  const handleAddContacts = async (contacts: SourceTopContact[]) => {
    for (const contact of contacts) {
      await createChannel.mutateAsync({
        type: contact.type as ChannelType,
        identifier: contact.identifier,
        name: contact.name,
      });
    }
  };

  const handleAddCustom = async (value: string) => {
    await addCustomSource.mutateAsync(value.trim());
  };

  const getTypeLabel = (_type: ChannelType) => {
    return 'Chat';
  };

  const getTypeColor = (_type: ChannelType) => {
    return colors.success;
  };

  const enabledChannels = channels?.filter((channel) => channel.enabled) ?? [];

  const renderChannelItem = ({ item }: { item: Channel }) => {
    // Show identifier only if it's different from the name
    const showIdentifier = item.name !== item.identifier;
    const showPushName = Boolean(item.push_name && item.push_name !== item.name);

    return (
      <View style={styles.channelItem}>
        <View style={styles.channelInfo}>
          <View style={styles.channelHeader}>
            <Text style={styles.channelName}>{item.name}</Text>
            <View style={[styles.typeBadge, { backgroundColor: getTypeColor(item.type) + '20' }]}>
              <Text style={[styles.typeText, { color: getTypeColor(item.type) }]}>
                {getTypeLabel(item.type)}
              </Text>
            </View>
          </View>
          {showPushName && (
            <Text style={styles.channelPushName}>{item.push_name}</Text>
          )}
          {showIdentifier && (
            <Text style={styles.channelIdentifier}>+{item.identifier}</Text>
          )}
        </View>
        <View style={styles.channelActions}>
          <Switch
            value={item.enabled}
            onValueChange={() => handleToggleChannel(item)}
            trackColor={{ false: colors.border, true: colors.primary }}
            thumbColor="#ffffff"
          />
          <TouchableOpacity style={styles.deleteButton} onPress={() => handleDeleteWhatsAppChannel(item)}>
            <Feather name="trash-2" size={18} color={colors.danger} />
          </TouchableOpacity>
        </View>
      </View>
    );
  };

  return (
    <View style={styles.screen}>
      <ScrollView style={styles.container} contentContainerStyle={styles.content}>
        <View style={styles.sectionHeader}>
          <Text style={styles.sectionTitle}>WhatsApp Chats</Text>
          <TouchableOpacity style={styles.addButton} onPress={handleOpenAddSourceModal}>
            <Feather name="plus" size={18} color={colors.primary} />
            <Text style={styles.addButtonText}>Add Chat</Text>
          </TouchableOpacity>
        </View>
        <Card>
          {channelsLoading ? (
            <LoadingSpinner />
          ) : enabledChannels.length > 0 ? (
            <FlatList
              data={enabledChannels}
              keyExtractor={(item) => String(item.id)}
              renderItem={renderChannelItem}
              scrollEnabled={false}
              ItemSeparatorComponent={() => <View style={styles.separator} />}
            />
          ) : (
            <View style={styles.emptyState}>
              <Feather name="message-circle" size={40} color={colors.textSecondary} />
              <Text style={styles.emptyStateText}>No WhatsApp chats selected</Text>
              <Text style={styles.emptyStateSubtext}>
                Add chats to track for events, reminders, and tasks
              </Text>
            </View>
          )}
        </Card>
      </ScrollView>

      <AddSourceModal
        visible={addSourceModalVisible}
        onClose={() => {
          setSearchQuery('');
          setAddSourceModalVisible(false);
        }}
        title="Add WhatsApp Chat"
        topContacts={topContacts}
        contactsLoading={contactsLoading}
        searchResults={searchResults}
        searchLoading={searchLoading}
        onSearchQueryChange={setSearchQuery}
        customInputPlaceholder="e.g. Levi Ackerman"
        customInputKeyboardType="default"
        validateCustomInput={validateChatName}
        blockManualAddWhenSearchResults
        onAddContacts={handleAddContacts}
        onAddCustom={handleAddCustom}
        addContactsLoading={createChannel.isPending}
        addCustomLoading={addCustomSource.isPending}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  screen: {
    flex: 1,
    backgroundColor: colors.background,
  },
  container: {
    flex: 1,
  },
  content: {
    padding: 16,
  },
  sectionHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
    marginLeft: 4,
  },
  sectionTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.textSecondary,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  addButton: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 12,
    paddingVertical: 6,
    backgroundColor: colors.primary + '15',
    borderRadius: 16,
  },
  addButtonText: {
    fontSize: 14,
    fontWeight: '500',
    color: colors.primary,
    marginLeft: 4,
  },
  channelItem: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 12,
  },
  channelInfo: {
    flex: 1,
    marginRight: 12,
  },
  channelHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 4,
  },
  channelName: {
    fontSize: 15,
    fontWeight: '500',
    color: colors.text,
    marginRight: 8,
  },
  channelPushName: {
    fontSize: 13,
    color: colors.textSecondary,
    marginBottom: 2,
  },
  typeBadge: {
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: 4,
  },
  typeText: {
    fontSize: 11,
    fontWeight: '600',
    textTransform: 'uppercase',
  },
  channelIdentifier: {
    fontSize: 13,
    color: colors.textSecondary,
  },
  channelActions: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  deleteButton: {
    marginLeft: 16,
    padding: 4,
  },
  separator: {
    height: 1,
    backgroundColor: colors.border,
  },
  emptyState: {
    alignItems: 'center',
    paddingVertical: 32,
  },
  emptyStateText: {
    fontSize: 16,
    fontWeight: '500',
    color: colors.text,
    marginTop: 12,
  },
  emptyStateSubtext: {
    fontSize: 13,
    color: colors.textSecondary,
    marginTop: 4,
    textAlign: 'center',
  },
});
