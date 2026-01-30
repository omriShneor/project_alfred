import React, { useState, useEffect } from 'react';
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
import { LoadingSpinner, Card, Button, Modal, Select } from '../../components/common';
import { colors } from '../../theme/colors';
import {
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

type DiscoveryTab = 'categories' | 'senders' | 'domains';

interface SelectedItem {
  type: EmailSourceType;
  identifier: string;
  name: string;
}

export function GmailPreferencesScreen() {
  const { data: gmailStatus } = useGmailStatus();

  const [addSourceModalVisible, setAddSourceModalVisible] = useState(false);
  const [activeDiscoveryTab, setActiveDiscoveryTab] = useState<DiscoveryTab>('categories');
  const [selectedItems, setSelectedItems] = useState<SelectedItem[]>([]);
  const [selectedCalendarId, setSelectedCalendarId] = useState<string>('');

  const { data: sources, isLoading: sourcesLoading } = useEmailSources();
  const googleConnected = gmailStatus?.connected ?? false;
  const { data: calendars } = useCalendars(googleConnected);

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

  useEffect(() => {
    if (calendars && calendars.length > 0 && !selectedCalendarId) {
      const primaryCalendar = calendars.find((c) => c.primary);
      setSelectedCalendarId(primaryCalendar?.id || calendars[0].id);
    }
  }, [calendars, selectedCalendarId]);

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
    // Check if Gmail is properly connected
    if (!gmailStatus?.connected || !gmailStatus?.has_scopes) {
      Alert.alert(
        'Gmail Not Connected',
        'Please reconnect your Google account to grant Gmail access.',
        [{ text: 'OK' }]
      );
      return;
    }

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

  return (
    <View style={styles.screen}>
      <ScrollView style={styles.container} contentContainerStyle={styles.content}>
        <View style={styles.sectionHeader}>
          <Text style={styles.sectionTitle}>Email Sources</Text>
          <TouchableOpacity
            style={styles.addButton}
            onPress={handleOpenAddSourceModal}
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
