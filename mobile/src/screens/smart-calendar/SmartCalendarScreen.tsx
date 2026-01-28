import React, { useState, useMemo, useEffect } from 'react';
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
import { SearchInput, FilterChips, LoadingSpinner, Card, Button, Modal, Select } from '../../components/common';
import { ChannelList, ChannelStats } from '../../components/channels';
import { colors } from '../../theme/colors';
import {
  useFeatures,
  useDiscoverableChannels,
  useDebounce,
  useWhatsAppStatus,
  useGCalStatus,
  useGmailStatus,
  useEmailSources,
  useCreateEmailSource,
  useUpdateEmailSource,
  useDeleteEmailSource,
  useDiscoverCategories,
  useDiscoverSenders,
  useDiscoverDomains,
  useCalendars,
} from '../../hooks';
import type {
  EmailSource,
  EmailSourceType,
  DiscoveredCategory,
  DiscoveredSender,
  DiscoveredDomain,
} from '../../types/gmail';

type TabType = 'channels' | 'email';
type DiscoveryTab = 'categories' | 'senders' | 'domains';

const TYPE_FILTERS = [
  { label: 'All', value: 'all' },
  { label: 'Contacts', value: 'sender' },
  { label: 'Groups', value: 'group' },
];

interface SelectedItem {
  type: EmailSourceType;
  identifier: string;
  name: string;
}

export function SmartCalendarScreen() {
  const { data: features } = useFeatures();
  const { data: waStatus } = useWhatsAppStatus();
  const { data: gcalStatus } = useGCalStatus();
  const { data: gmailStatus } = useGmailStatus();

  // Determine which tabs to show
  const showChannelsTab = features?.smart_calendar?.inputs?.whatsapp?.enabled ?? false;
  const showEmailTab = features?.smart_calendar?.inputs?.email?.enabled ?? false;

  // Set initial tab based on enabled inputs
  const [activeTab, setActiveTab] = useState<TabType>(showChannelsTab ? 'channels' : 'email');

  // Channels state
  const [searchQuery, setSearchQuery] = useState('');
  const [typeFilter, setTypeFilter] = useState('all');
  const debouncedSearch = useDebounce(searchQuery, 150);

  // Only fetch channels when WhatsApp is connected
  const {
    data: channels,
    isLoading: channelsLoading,
    refetch: refetchChannels,
    isRefetching: isRefetchingChannels,
  } = useDiscoverableChannels({ enabled: waStatus?.connected ?? false });

  // Email sources state
  const [addSourceModalVisible, setAddSourceModalVisible] = useState(false);
  const [activeDiscoveryTab, setActiveDiscoveryTab] = useState<DiscoveryTab>('categories');
  const [selectedItems, setSelectedItems] = useState<SelectedItem[]>([]);
  const [selectedCalendarId, setSelectedCalendarId] = useState<string>('');

  const { data: sources, isLoading: sourcesLoading } = useEmailSources();
  const { data: calendars } = useCalendars();

  const createSource = useCreateEmailSource();
  const updateSource = useUpdateEmailSource();
  const deleteSource = useDeleteEmailSource();

  const {
    data: categories,
    isLoading: categoriesLoading,
    refetch: refetchCategories,
  } = useDiscoverCategories();
  const {
    data: senders,
    isLoading: sendersLoading,
    refetch: refetchSenders,
  } = useDiscoverSenders(50);
  const {
    data: domains,
    isLoading: domainsLoading,
    refetch: refetchDomains,
  } = useDiscoverDomains(50);

  // Filter channels
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

  // Set default calendar when calendars load
  useEffect(() => {
    if (calendars && calendars.length > 0 && !selectedCalendarId) {
      const primaryCalendar = calendars.find((c) => c.primary);
      setSelectedCalendarId(primaryCalendar?.id || calendars[0].id);
    }
  }, [calendars, selectedCalendarId]);

  // Email source handlers
  const handleToggleSource = async (source: EmailSource) => {
    try {
      await updateSource.mutateAsync({
        id: source.id,
        data: { enabled: !source.enabled },
      });
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to update source');
    }
  };

  const handleDeleteSource = (source: EmailSource) => {
    Alert.alert(
      'Delete Source',
      `Are you sure you want to delete "${source.name}"?`,
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Delete',
          style: 'destructive',
          onPress: async () => {
            try {
              await deleteSource.mutateAsync(source.id);
            } catch (error: any) {
              Alert.alert('Error', error.message || 'Failed to delete source');
            }
          },
        },
      ]
    );
  };

  const handleOpenAddSourceModal = () => {
    setSelectedItems([]);
    setActiveDiscoveryTab('categories');
    setAddSourceModalVisible(true);
    refetchCategories();
    refetchSenders();
    refetchDomains();
  };

  const toggleItemSelection = (item: SelectedItem) => {
    setSelectedItems((prev) => {
      const exists = prev.some(
        (i) => i.type === item.type && i.identifier === item.identifier
      );
      if (exists) {
        return prev.filter(
          (i) => !(i.type === item.type && i.identifier === item.identifier)
        );
      }
      return [...prev, item];
    });
  };

  const isItemSelected = (type: EmailSourceType, identifier: string) => {
    return selectedItems.some(
      (i) => i.type === type && i.identifier === identifier
    );
  };

  const isSourceAlreadyAdded = (type: EmailSourceType, identifier: string) => {
    return sources?.some(
      (s) => s.type === type && s.identifier === identifier
    );
  };

  const handleAddSelectedSources = async () => {
    if (selectedItems.length === 0) {
      Alert.alert('Error', 'Please select at least one source to add');
      return;
    }
    if (!selectedCalendarId) {
      Alert.alert('Error', 'Please select a calendar');
      return;
    }

    try {
      for (const item of selectedItems) {
        await createSource.mutateAsync({
          type: item.type,
          identifier: item.identifier,
          name: item.name,
          calendar_id: selectedCalendarId,
        });
      }
      setAddSourceModalVisible(false);
      setSelectedItems([]);
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to add sources');
    }
  };

  const getSourceTypeLabel = (type: EmailSourceType) => {
    switch (type) {
      case 'category': return 'Category';
      case 'sender': return 'Sender';
      case 'domain': return 'Domain';
      default: return type;
    }
  };

  const getSourceTypeColor = (type: EmailSourceType) => {
    switch (type) {
      case 'category': return colors.primary;
      case 'sender': return colors.success;
      case 'domain': return colors.warning;
      default: return colors.textSecondary;
    }
  };

  const renderCategoryItem = ({ item }: { item: DiscoveredCategory }) => {
    const alreadyAdded = isSourceAlreadyAdded('category', item.id);
    const selected = isItemSelected('category', item.id);
    return (
      <TouchableOpacity
        style={[styles.discoveryItem, selected && styles.discoveryItemSelected, alreadyAdded && styles.discoveryItemDisabled]}
        onPress={() => !alreadyAdded && toggleItemSelection({ type: 'category', identifier: item.id, name: item.name })}
        disabled={alreadyAdded}
      >
        <View style={styles.discoveryItemContent}>
          <Text style={styles.discoveryItemName}>{item.name}</Text>
          <Text style={styles.discoveryItemDescription}>{item.description}</Text>
          <Text style={styles.discoveryItemCount}>{item.email_count} emails</Text>
        </View>
        {alreadyAdded ? (
          <Feather name="check-circle" size={20} color={colors.success} />
        ) : selected ? (
          <Feather name="check-square" size={20} color={colors.primary} />
        ) : (
          <Feather name="square" size={20} color={colors.textSecondary} />
        )}
      </TouchableOpacity>
    );
  };

  const renderSenderItem = ({ item }: { item: DiscoveredSender }) => {
    const alreadyAdded = isSourceAlreadyAdded('sender', item.email);
    const selected = isItemSelected('sender', item.email);
    return (
      <TouchableOpacity
        style={[styles.discoveryItem, selected && styles.discoveryItemSelected, alreadyAdded && styles.discoveryItemDisabled]}
        onPress={() => !alreadyAdded && toggleItemSelection({ type: 'sender', identifier: item.email, name: item.name || item.email })}
        disabled={alreadyAdded}
      >
        <View style={styles.discoveryItemContent}>
          <Text style={styles.discoveryItemName}>{item.name || item.email}</Text>
          {item.name && <Text style={styles.discoveryItemDescription}>{item.email}</Text>}
          <Text style={styles.discoveryItemCount}>{item.email_count} emails</Text>
        </View>
        {alreadyAdded ? (
          <Feather name="check-circle" size={20} color={colors.success} />
        ) : selected ? (
          <Feather name="check-square" size={20} color={colors.primary} />
        ) : (
          <Feather name="square" size={20} color={colors.textSecondary} />
        )}
      </TouchableOpacity>
    );
  };

  const renderDomainItem = ({ item }: { item: DiscoveredDomain }) => {
    const alreadyAdded = isSourceAlreadyAdded('domain', item.domain);
    const selected = isItemSelected('domain', item.domain);
    return (
      <TouchableOpacity
        style={[styles.discoveryItem, selected && styles.discoveryItemSelected, alreadyAdded && styles.discoveryItemDisabled]}
        onPress={() => !alreadyAdded && toggleItemSelection({ type: 'domain', identifier: item.domain, name: item.domain })}
        disabled={alreadyAdded}
      >
        <View style={styles.discoveryItemContent}>
          <Text style={styles.discoveryItemName}>{item.domain}</Text>
          <Text style={styles.discoveryItemCount}>{item.email_count} emails</Text>
        </View>
        {alreadyAdded ? (
          <Feather name="check-circle" size={20} color={colors.success} />
        ) : selected ? (
          <Feather name="check-square" size={20} color={colors.primary} />
        ) : (
          <Feather name="square" size={20} color={colors.textSecondary} />
        )}
      </TouchableOpacity>
    );
  };

  const renderSourceItem = ({ item }: { item: EmailSource }) => (
    <View style={styles.sourceItem}>
      <View style={styles.sourceInfo}>
        <View style={styles.sourceHeader}>
          <Text style={styles.sourceName}>{item.name}</Text>
          <View style={[styles.sourceTypeBadge, { backgroundColor: getSourceTypeColor(item.type) + '20' }]}>
            <Text style={[styles.sourceTypeText, { color: getSourceTypeColor(item.type) }]}>
              {getSourceTypeLabel(item.type)}
            </Text>
          </View>
        </View>
        <Text style={styles.sourceIdentifier}>{item.identifier}</Text>
      </View>
      <View style={styles.sourceActions}>
        <Switch
          value={item.enabled}
          onValueChange={() => handleToggleSource(item)}
          trackColor={{ false: colors.border, true: colors.primary }}
          thumbColor="#ffffff"
        />
        <TouchableOpacity style={styles.deleteButton} onPress={() => handleDeleteSource(item)}>
          <Feather name="trash-2" size={18} color={colors.danger} />
        </TouchableOpacity>
      </View>
    </View>
  );

  const calendarOptions = calendars?.map((c) => ({
    label: c.summary + (c.primary ? ' (Primary)' : ''),
    value: c.id,
  })) || [];

  const isDiscoveryLoading =
    (activeDiscoveryTab === 'categories' && categoriesLoading) ||
    (activeDiscoveryTab === 'senders' && sendersLoading) ||
    (activeDiscoveryTab === 'domains' && domainsLoading);

  const currentDiscoveryData =
    activeDiscoveryTab === 'categories' ? categories :
    activeDiscoveryTab === 'senders' ? senders : domains;

  // Render tabs only if both inputs are enabled
  const showTabs = showChannelsTab && showEmailTab;

  return (
    <View style={styles.screen}>
      {/* Tabs (only if both inputs enabled) */}
      {showTabs && (
        <View style={styles.tabBar}>
          <TouchableOpacity
            style={[styles.tab, activeTab === 'channels' && styles.tabActive]}
            onPress={() => setActiveTab('channels')}
          >
            <Feather name="message-circle" size={16} color={activeTab === 'channels' ? colors.primary : colors.textSecondary} />
            <Text style={[styles.tabText, activeTab === 'channels' && styles.tabTextActive]}>Channels</Text>
          </TouchableOpacity>
          <TouchableOpacity
            style={[styles.tab, activeTab === 'email' && styles.tabActive]}
            onPress={() => setActiveTab('email')}
          >
            <Feather name="mail" size={16} color={activeTab === 'email' ? colors.primary : colors.textSecondary} />
            <Text style={[styles.tabText, activeTab === 'email' && styles.tabTextActive]}>Email Sources</Text>
          </TouchableOpacity>
        </View>
      )}

      {/* Content */}
      <View style={styles.content}>
        {(activeTab === 'channels' && showChannelsTab) ? (
          // Channels Tab
          <View style={styles.channelsContainer}>
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
            {channelsLoading ? (
              <LoadingSpinner message="Loading channels..." />
            ) : (
              <ChannelList
                channels={filteredChannels}
                refreshing={isRefetchingChannels}
                onRefresh={refetchChannels}
              />
            )}
          </View>
        ) : showEmailTab ? (
          // Email Sources Tab
          <ScrollView style={styles.emailContainer} contentContainerStyle={styles.emailContent}>
            <View style={styles.sectionHeader}>
              <Text style={styles.sectionTitle}>Email Sources</Text>
              <TouchableOpacity
                style={styles.addButton}
                onPress={handleOpenAddSourceModal}
                disabled={!gmailStatus?.connected || !gmailStatus?.has_scopes}
              >
                <Feather name="plus" size={18} color={colors.primary} />
                <Text style={styles.addButtonText}>Add Source</Text>
              </TouchableOpacity>
            </View>
            <Card>
              {sourcesLoading ? (
                <LoadingSpinner />
              ) : sources && sources.length > 0 ? (
                <FlatList
                  data={sources}
                  keyExtractor={(item) => String(item.id)}
                  renderItem={renderSourceItem}
                  scrollEnabled={false}
                  ItemSeparatorComponent={() => <View style={styles.separator} />}
                />
              ) : (
                <View style={styles.emptyState}>
                  <Feather name="inbox" size={40} color={colors.textSecondary} />
                  <Text style={styles.emptyStateText}>No email sources configured</Text>
                  <Text style={styles.emptyStateSubtext}>
                    Add categories, senders, or domains to track for events
                  </Text>
                </View>
              )}
            </Card>
          </ScrollView>
        ) : null}
      </View>

      {/* Status Section */}
      <View style={styles.statusSection}>
        <Text style={styles.statusTitle}>Status</Text>
        <View style={styles.statusRow}>
          {showChannelsTab && (
            <View style={styles.statusItem}>
              <View style={[styles.statusDot, { backgroundColor: waStatus?.connected ? colors.success : colors.danger }]} />
              <Text style={styles.statusLabel}>WhatsApp</Text>
            </View>
          )}
          <View style={styles.statusItem}>
            <View style={[styles.statusDot, { backgroundColor: gcalStatus?.connected ? colors.success : colors.danger }]} />
            <Text style={styles.statusLabel}>Google Calendar</Text>
          </View>
          {showEmailTab && (
            <View style={styles.statusItem}>
              <View style={[styles.statusDot, { backgroundColor: gmailStatus?.connected && gmailStatus?.has_scopes ? colors.success : colors.danger }]} />
              <Text style={styles.statusLabel}>Gmail</Text>
            </View>
          )}
        </View>
      </View>

      {/* Add Source Modal */}
      <Modal
        visible={addSourceModalVisible}
        onClose={() => setAddSourceModalVisible(false)}
        title="Add Email Source"
      >
        <View style={styles.modalTabContainer}>
          <TouchableOpacity
            style={[styles.modalTab, activeDiscoveryTab === 'categories' && styles.modalTabActive]}
            onPress={() => setActiveDiscoveryTab('categories')}
          >
            <Text style={[styles.modalTabText, activeDiscoveryTab === 'categories' && styles.modalTabTextActive]}>Categories</Text>
          </TouchableOpacity>
          <TouchableOpacity
            style={[styles.modalTab, activeDiscoveryTab === 'senders' && styles.modalTabActive]}
            onPress={() => setActiveDiscoveryTab('senders')}
          >
            <Text style={[styles.modalTabText, activeDiscoveryTab === 'senders' && styles.modalTabTextActive]}>Senders</Text>
          </TouchableOpacity>
          <TouchableOpacity
            style={[styles.modalTab, activeDiscoveryTab === 'domains' && styles.modalTabActive]}
            onPress={() => setActiveDiscoveryTab('domains')}
          >
            <Text style={[styles.modalTabText, activeDiscoveryTab === 'domains' && styles.modalTabTextActive]}>Domains</Text>
          </TouchableOpacity>
        </View>

        <View style={styles.calendarSection}>
          <Text style={styles.calendarLabel}>Target Calendar</Text>
          <Select
            options={calendarOptions}
            value={selectedCalendarId}
            onChange={setSelectedCalendarId}
            placeholder="Select calendar"
          />
        </View>

        <View style={styles.discoveryList}>
          {isDiscoveryLoading ? (
            <LoadingSpinner />
          ) : !currentDiscoveryData || currentDiscoveryData.length === 0 ? (
            <View style={styles.emptyDiscovery}>
              <Text style={styles.emptyDiscoveryText}>No {activeDiscoveryTab} found</Text>
            </View>
          ) : (
            <FlatList
              data={currentDiscoveryData as any[]}
              keyExtractor={(item, index) =>
                activeDiscoveryTab === 'categories' ? (item as DiscoveredCategory).id :
                activeDiscoveryTab === 'senders' ? (item as DiscoveredSender).email :
                (item as DiscoveredDomain).domain
              }
              renderItem={
                activeDiscoveryTab === 'categories' ? (renderCategoryItem as any) :
                activeDiscoveryTab === 'senders' ? (renderSenderItem as any) :
                (renderDomainItem as any)
              }
              style={styles.discoveryFlatList}
              ItemSeparatorComponent={() => <View style={styles.separator} />}
            />
          )}
        </View>

        <View style={styles.modalFooter}>
          <Button
            title={`Add ${selectedItems.length} Source${selectedItems.length !== 1 ? 's' : ''}`}
            onPress={handleAddSelectedSources}
            loading={createSource.isPending}
            disabled={selectedItems.length === 0}
          />
        </View>
      </Modal>
    </View>
  );
}

const styles = StyleSheet.create({
  screen: {
    flex: 1,
    backgroundColor: colors.background,
  },
  tabBar: {
    flexDirection: 'row',
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
    backgroundColor: colors.card,
  },
  tab: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 12,
    gap: 6,
  },
  tabActive: {
    borderBottomWidth: 2,
    borderBottomColor: colors.primary,
  },
  tabText: {
    fontSize: 14,
    fontWeight: '500',
    color: colors.textSecondary,
  },
  tabTextActive: {
    color: colors.primary,
  },
  content: {
    flex: 1,
  },
  channelsContainer: {
    flex: 1,
    padding: 16,
  },
  emailContainer: {
    flex: 1,
  },
  emailContent: {
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
  sourceItem: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 12,
  },
  sourceInfo: {
    flex: 1,
    marginRight: 12,
  },
  sourceHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 4,
  },
  sourceName: {
    fontSize: 15,
    fontWeight: '500',
    color: colors.text,
    marginRight: 8,
  },
  sourceTypeBadge: {
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: 4,
  },
  sourceTypeText: {
    fontSize: 11,
    fontWeight: '600',
    textTransform: 'uppercase',
  },
  sourceIdentifier: {
    fontSize: 13,
    color: colors.textSecondary,
  },
  sourceActions: {
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
  statusSection: {
    borderTopWidth: 1,
    borderTopColor: colors.border,
    padding: 16,
    backgroundColor: colors.card,
  },
  statusTitle: {
    fontSize: 12,
    fontWeight: '600',
    color: colors.textSecondary,
    textTransform: 'uppercase',
    marginBottom: 8,
  },
  statusRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 16,
  },
  statusItem: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  statusDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    marginRight: 6,
  },
  statusLabel: {
    fontSize: 13,
    color: colors.text,
  },
  modalTabContainer: {
    flexDirection: 'row',
    borderRadius: 8,
    backgroundColor: colors.background,
    padding: 4,
    marginBottom: 16,
  },
  modalTab: {
    flex: 1,
    paddingVertical: 8,
    alignItems: 'center',
    borderRadius: 6,
  },
  modalTabActive: {
    backgroundColor: colors.card,
  },
  modalTabText: {
    fontSize: 14,
    fontWeight: '500',
    color: colors.textSecondary,
  },
  modalTabTextActive: {
    color: colors.primary,
  },
  calendarSection: {
    marginBottom: 16,
  },
  calendarLabel: {
    fontSize: 14,
    fontWeight: '500',
    color: colors.text,
    marginBottom: 8,
  },
  discoveryList: {
    maxHeight: 300,
    marginBottom: 16,
  },
  discoveryFlatList: {
    maxHeight: 300,
  },
  discoveryItem: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 12,
    paddingHorizontal: 8,
    borderRadius: 8,
  },
  discoveryItemSelected: {
    backgroundColor: colors.primary + '10',
  },
  discoveryItemDisabled: {
    opacity: 0.5,
  },
  discoveryItemContent: {
    flex: 1,
    marginRight: 12,
  },
  discoveryItemName: {
    fontSize: 15,
    fontWeight: '500',
    color: colors.text,
  },
  discoveryItemDescription: {
    fontSize: 13,
    color: colors.textSecondary,
    marginTop: 2,
  },
  discoveryItemCount: {
    fontSize: 12,
    color: colors.textSecondary,
    marginTop: 2,
  },
  emptyDiscovery: {
    alignItems: 'center',
    paddingVertical: 24,
  },
  emptyDiscoveryText: {
    fontSize: 14,
    color: colors.textSecondary,
  },
  modalFooter: {
    paddingTop: 16,
    borderTopWidth: 1,
    borderTopColor: colors.border,
  },
});
