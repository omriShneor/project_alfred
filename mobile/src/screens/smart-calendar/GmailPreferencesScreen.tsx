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
  TextInput,
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
  useTopContacts,
  useAddCustomSource,
  useCalendars,
} from '../../hooks';
import type { EmailSource, EmailSourceType, TopContact } from '../../types/gmail';
import type { Calendar } from '../../types/event';

export function GmailPreferencesScreen() {
  const { data: gmailStatus } = useGmailStatus();

  const [addSourceModalVisible, setAddSourceModalVisible] = useState(false);
  const [selectedContacts, setSelectedContacts] = useState<Set<string>>(new Set());
  const [customInput, setCustomInput] = useState('');
  const [customInputError, setCustomInputError] = useState<string | null>(null);
  const [selectedCalendarId, setSelectedCalendarId] = useState<string>('');

  const { data: sources, isLoading: sourcesLoading } = useEmailSources();
  const googleConnected = gmailStatus?.connected ?? false;
  const { data: calendars } = useCalendars(googleConnected);
  const { data: topContacts, isLoading: contactsLoading } = useTopContacts();

  const createSource = useCreateEmailSource();
  const updateSource = useUpdateEmailSource();
  const deleteSource = useDeleteEmailSource();
  const addCustomSource = useAddCustomSource();

  useEffect(() => {
    if (calendars && calendars.length > 0 && !selectedCalendarId) {
      const primaryCalendar = calendars.find((c: Calendar) => c.primary);
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
    if (!gmailStatus?.connected || !gmailStatus?.has_scopes) {
      Alert.alert(
        'Gmail Not Connected',
        'Please reconnect your Google account to grant Gmail access.',
        [{ text: 'OK' }]
      );
      return;
    }

    setSelectedContacts(new Set());
    setCustomInput('');
    setCustomInputError(null);
    setAddSourceModalVisible(true);
  };

  const toggleContactSelection = (email: string) => {
    setSelectedContacts((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(email)) {
        newSet.delete(email);
      } else {
        newSet.add(email);
      }
      return newSet;
    });
  };

  const validateCustomInput = (value: string): string | null => {
    if (!value.trim()) return null;
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    const domainRegex = /^@?[a-zA-Z0-9][a-zA-Z0-9.-]*\.[a-zA-Z]{2,}$/;
    if (emailRegex.test(value) || domainRegex.test(value)) {
      return null;
    }
    return 'Enter a valid email (user@domain.com) or domain (domain.com)';
  };

  const handleAddCustom = async () => {
    const error = validateCustomInput(customInput);
    if (error) {
      setCustomInputError(error);
      return;
    }

    if (!selectedCalendarId) {
      Alert.alert('Error', 'Please select a calendar');
      return;
    }

    try {
      await addCustomSource.mutateAsync({
        value: customInput.trim(),
        calendar_id: selectedCalendarId,
      });
      setCustomInput('');
      setCustomInputError(null);
      setAddSourceModalVisible(false);
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to add source');
    }
  };

  const handleAddSelectedContacts = async () => {
    if (selectedContacts.size === 0) {
      Alert.alert('Error', 'Please select at least one contact');
      return;
    }

    if (!selectedCalendarId) {
      Alert.alert('Error', 'Please select a calendar');
      return;
    }

    try {
      for (const email of selectedContacts) {
        const contact = topContacts?.find((c: TopContact) => c.email === email);
        await createSource.mutateAsync({
          type: 'sender',
          identifier: email,
          name: contact?.name || email,
          calendar_id: selectedCalendarId,
        });
      }
      setSelectedContacts(new Set());
      setAddSourceModalVisible(false);
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

  const renderContactItem = ({ item }: { item: TopContact }) => {
    const selected = selectedContacts.has(item.email);
    return (
      <TouchableOpacity
        style={[styles.contactItem, selected && styles.contactItemSelected, item.is_tracked && styles.contactItemDisabled]}
        onPress={() => !item.is_tracked && toggleContactSelection(item.email)}
        disabled={item.is_tracked}
      >
        <View style={styles.contactInfo}>
          <Text style={styles.contactName} numberOfLines={1}>
            {item.name || item.email}
          </Text>
          {item.name && (
            <Text style={styles.contactEmail} numberOfLines={1}>
              {item.email}
            </Text>
          )}
          <Text style={styles.contactCount}>{item.email_count} emails</Text>
        </View>
        {item.is_tracked ? (
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

  const calendarOptions = calendars?.map((c: Calendar) => ({
    label: c.summary + (c.primary ? ' (Primary)' : ''),
    value: c.id,
  })) || [];

  // Filter out already tracked contacts
  const availableContacts = topContacts?.filter((c) => !c.is_tracked) || [];
  const hasSelectableContacts = availableContacts.length > 0;

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
                Add contacts or domains to track for events
              </Text>
            </View>
          )}
        </Card>
      </ScrollView>

      <Modal
        visible={addSourceModalVisible}
        onClose={() => setAddSourceModalVisible(false)}
        title="Add Email Source"
        scrollable={false}
      >
        <View style={styles.calendarSection}>
          <Text style={styles.calendarLabel}>Target Calendar</Text>
          <Select
            options={calendarOptions}
            value={selectedCalendarId}
            onChange={setSelectedCalendarId}
            placeholder="Select calendar"
          />
        </View>

        {/* Suggested Contacts Section */}
        <View style={styles.suggestedSection}>
          <Text style={styles.sectionLabel}>Suggested Contacts</Text>
          {contactsLoading ? (
            <LoadingSpinner />
          ) : topContacts && topContacts.length > 0 ? (
            <FlatList
              data={topContacts}
              keyExtractor={(item) => item.email}
              renderItem={renderContactItem}
              style={styles.contactList}
              ItemSeparatorComponent={() => <View style={styles.separator} />}
            />
          ) : (
            <View style={styles.emptyContacts}>
              <Text style={styles.emptyContactsText}>No contacts found</Text>
              <Text style={styles.emptyContactsSubtext}>
                Add a custom email or domain below
              </Text>
            </View>
          )}

          {hasSelectableContacts && selectedContacts.size > 0 && (
            <Button
              title={`Add ${selectedContacts.size} Contact${selectedContacts.size !== 1 ? 's' : ''}`}
              onPress={handleAddSelectedContacts}
              loading={createSource.isPending}
              style={styles.addContactsButton}
            />
          )}
        </View>

        {/* Custom Input Section */}
        <View style={styles.customSection}>
          <Text style={styles.sectionLabel}>Or add a custom email/domain</Text>
          <TextInput
            style={[styles.customInput, customInputError && styles.customInputError]}
            value={customInput}
            onChangeText={(text) => {
              setCustomInput(text);
              if (customInputError) setCustomInputError(null);
            }}
            placeholder="e.g. boss@work.com or acme.com"
            placeholderTextColor={colors.textSecondary}
            keyboardType="email-address"
            autoCapitalize="none"
            autoCorrect={false}
            onBlur={() => {
              if (customInput.trim()) {
                setCustomInputError(validateCustomInput(customInput));
              }
            }}
          />
          {customInputError && (
            <Text style={styles.errorText}>{customInputError}</Text>
          )}
          <Button
            title="Add Custom"
            variant="outline"
            onPress={handleAddCustom}
            loading={addCustomSource.isPending}
            disabled={!customInput.trim()}
            style={styles.addCustomButton}
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
  calendarSection: {
    marginBottom: 16,
  },
  calendarLabel: {
    fontSize: 14,
    fontWeight: '500',
    color: colors.text,
    marginBottom: 8,
  },
  sectionLabel: {
    fontSize: 14,
    fontWeight: '500',
    color: colors.text,
    marginBottom: 12,
  },
  suggestedSection: {
    marginBottom: 16,
  },
  contactList: {
    maxHeight: 200,
  },
  contactItem: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 10,
    paddingHorizontal: 8,
    borderRadius: 8,
  },
  contactItemSelected: {
    backgroundColor: colors.primary + '10',
  },
  contactItemDisabled: {
    opacity: 0.5,
  },
  contactInfo: {
    flex: 1,
    marginRight: 12,
  },
  contactName: {
    fontSize: 15,
    fontWeight: '500',
    color: colors.text,
  },
  contactEmail: {
    fontSize: 13,
    color: colors.textSecondary,
    marginTop: 2,
  },
  contactCount: {
    fontSize: 12,
    color: colors.textSecondary,
    marginTop: 2,
  },
  emptyContacts: {
    alignItems: 'center',
    paddingVertical: 16,
  },
  emptyContactsText: {
    fontSize: 14,
    color: colors.textSecondary,
  },
  emptyContactsSubtext: {
    fontSize: 12,
    color: colors.textSecondary,
    marginTop: 4,
  },
  addContactsButton: {
    marginTop: 12,
  },
  customSection: {
    borderTopWidth: 1,
    borderTopColor: colors.border,
    paddingTop: 16,
  },
  customInput: {
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 8,
    paddingHorizontal: 12,
    paddingVertical: 10,
    fontSize: 15,
    color: colors.text,
    backgroundColor: colors.background,
  },
  customInputError: {
    borderColor: colors.danger,
  },
  errorText: {
    fontSize: 12,
    color: colors.danger,
    marginTop: 4,
  },
  addCustomButton: {
    marginTop: 12,
  },
});
