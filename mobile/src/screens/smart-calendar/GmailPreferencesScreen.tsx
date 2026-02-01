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
  Keyboard,
  Platform,
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

interface CustomEntry {
  identifier: string;
  name: string;
}

export function GmailPreferencesScreen() {
  const { data: gmailStatus } = useGmailStatus();

  const [addSourceModalVisible, setAddSourceModalVisible] = useState(false);
  const [selectedContacts, setSelectedContacts] = useState<Set<string>>(new Set());
  const [customEntries, setCustomEntries] = useState<CustomEntry[]>([]);
  const [customInput, setCustomInput] = useState('');
  const [customInputError, setCustomInputError] = useState<string | null>(null);
  const [selectedCalendarId, setSelectedCalendarId] = useState<string>('');
  const [isCustomInputFocused, setIsCustomInputFocused] = useState(false);
  const [keyboardHeight, setKeyboardHeight] = useState(0);
  const [isAdding, setIsAdding] = useState(false);

  // Track keyboard visibility
  useEffect(() => {
    const showEvent = Platform.OS === 'ios' ? 'keyboardWillShow' : 'keyboardDidShow';
    const hideEvent = Platform.OS === 'ios' ? 'keyboardWillHide' : 'keyboardDidHide';

    const showSubscription = Keyboard.addListener(showEvent, (e) => {
      setKeyboardHeight(e.endCoordinates.height);
    });
    const hideSubscription = Keyboard.addListener(hideEvent, () => {
      setKeyboardHeight(0);
      setIsCustomInputFocused(false);
    });

    return () => {
      showSubscription.remove();
      hideSubscription.remove();
    };
  }, []);

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
    setCustomEntries([]);
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

  const removeCustomEntry = (identifier: string) => {
    setCustomEntries((prev) => prev.filter((e) => e.identifier !== identifier));
    setSelectedContacts((prev) => {
      const newSet = new Set(prev);
      newSet.delete(identifier);
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

  const handleAddCustomToList = () => {
    const error = validateCustomInput(customInput);
    if (error) {
      setCustomInputError(error);
      return;
    }

    const trimmedValue = customInput.trim();

    // Check if already in custom entries
    if (customEntries.some((e) => e.identifier === trimmedValue)) {
      setCustomInputError('Already added');
      return;
    }

    // Check if already in top contacts
    if (topContacts?.some((c: TopContact) => c.email === trimmedValue)) {
      setCustomInputError('Already in suggested contacts');
      return;
    }

    // Add to custom entries and select it
    setCustomEntries((prev) => [...prev, { identifier: trimmedValue, name: trimmedValue }]);
    setSelectedContacts((prev) => new Set(prev).add(trimmedValue));
    setCustomInput('');
    setCustomInputError(null);
    Keyboard.dismiss();
  };

  const handleAddAllSelected = async () => {
    if (!selectedCalendarId) {
      Alert.alert('Error', 'Please select a calendar');
      return;
    }

    setIsAdding(true);
    try {
      // Add selected top contacts
      for (const email of selectedContacts) {
        const contact = topContacts?.find((c: TopContact) => c.email === email);
        if (contact && !contact.is_tracked) {
          await createSource.mutateAsync({
            type: 'sender',
            identifier: email,
            name: contact.name || email,
            calendar_id: selectedCalendarId,
          });
        }
      }

      // Add custom entries
      const customToAdd = customEntries.filter((e) => selectedContacts.has(e.identifier));
      for (const entry of customToAdd) {
        await addCustomSource.mutateAsync({
          value: entry.identifier,
          calendar_id: selectedCalendarId,
        });
      }

      setSelectedContacts(new Set());
      setCustomEntries([]);
      setAddSourceModalVisible(false);
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to add sources');
    } finally {
      setIsAdding(false);
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

  // Total selected count
  const totalSelected = selectedContacts.size;

  const renderContactItem = (item: TopContact, index: number) => {
    const selected = selectedContacts.has(item.email);
    return (
      <React.Fragment key={item.email}>
        {index > 0 && <View style={styles.separator} />}
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
      </React.Fragment>
    );
  };

  const renderCustomEntry = (entry: CustomEntry, index: number) => {
    const selected = selectedContacts.has(entry.identifier);
    return (
      <React.Fragment key={`custom-${entry.identifier}`}>
        {index > 0 && <View style={styles.separator} />}
        <TouchableOpacity
          style={[styles.contactItem, selected && styles.contactItemSelected]}
          onPress={() => toggleContactSelection(entry.identifier)}
        >
          <View style={styles.contactInfo}>
            <View style={styles.customEntryHeader}>
              <Text style={styles.contactName} numberOfLines={1}>
                {entry.identifier}
              </Text>
              <View style={styles.customBadge}>
                <Text style={styles.customBadgeText}>Custom</Text>
              </View>
            </View>
          </View>
          <View style={styles.customEntryActions}>
            {selected ? (
              <Feather name="check-square" size={20} color={colors.primary} />
            ) : (
              <Feather name="square" size={20} color={colors.textSecondary} />
            )}
            <TouchableOpacity
              style={styles.removeButton}
              onPress={() => removeCustomEntry(entry.identifier)}
            >
              <Feather name="x" size={18} color={colors.textSecondary} />
            </TouchableOpacity>
          </View>
        </TouchableOpacity>
      </React.Fragment>
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
        footer={
          isCustomInputFocused && keyboardHeight > 0 ? (
            <View style={styles.floatingFooter}>
              <Button
                title="Add to List"
                onPress={handleAddCustomToList}
                disabled={!customInput.trim()}
              />
            </View>
          ) : totalSelected > 0 ? (
            <View style={styles.floatingFooter}>
              <Button
                title={`Add ${totalSelected} Source${totalSelected !== 1 ? 's' : ''}`}
                onPress={handleAddAllSelected}
                loading={isAdding || createSource.isPending || addCustomSource.isPending}
              />
            </View>
          ) : undefined
        }
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

        {/* Custom Entries Section (shown at top if any) */}
        {customEntries.length > 0 && (
          <View style={styles.customEntriesSection}>
            <Text style={styles.sectionLabel}>Custom Entries</Text>
            <View style={styles.contactList}>
              {customEntries.map((entry, index) => renderCustomEntry(entry, index))}
            </View>
          </View>
        )}

        {/* Suggested Contacts Section */}
        <View style={styles.suggestedSection}>
          <Text style={styles.sectionLabel}>Suggested Contacts</Text>
          {contactsLoading ? (
            <LoadingSpinner />
          ) : topContacts && topContacts.length > 0 ? (
            <View style={styles.contactList}>
              {topContacts.map((item, index) => renderContactItem(item, index))}
            </View>
          ) : (
            <View style={styles.emptyContacts}>
              <Text style={styles.emptyContactsText}>No contacts found</Text>
              <Text style={styles.emptyContactsSubtext}>
                Add a custom email or domain below
              </Text>
            </View>
          )}
        </View>

        {/* Custom Input Section */}
        <View style={styles.customSection}>
          <Text style={styles.sectionLabel}>Add manually</Text>
          <View style={styles.customInputRow}>
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
              onFocus={() => setIsCustomInputFocused(true)}
              onBlur={() => {
                if (customInput.trim()) {
                  setCustomInputError(validateCustomInput(customInput));
                }
              }}
            />
            {!isCustomInputFocused && customInput.trim() && (
              <TouchableOpacity
                style={styles.addToListButton}
                onPress={handleAddCustomToList}
              >
                <Feather name="plus" size={20} color={colors.primary} />
              </TouchableOpacity>
            )}
          </View>
          {customInputError && (
            <Text style={styles.errorText}>{customInputError}</Text>
          )}
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
  customEntriesSection: {
    marginBottom: 16,
  },
  suggestedSection: {
    marginBottom: 16,
  },
  contactList: {
    // No maxHeight - let the Modal's ScrollView handle scrolling
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
  customEntryHeader: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  customBadge: {
    marginLeft: 8,
    backgroundColor: colors.primary + '20',
    paddingHorizontal: 6,
    paddingVertical: 2,
    borderRadius: 4,
  },
  customBadgeText: {
    fontSize: 10,
    fontWeight: '600',
    color: colors.primary,
    textTransform: 'uppercase',
  },
  customEntryActions: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  removeButton: {
    marginLeft: 12,
    padding: 4,
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
  customSection: {
    borderTopWidth: 1,
    borderTopColor: colors.border,
    paddingTop: 16,
  },
  customInputRow: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  customInput: {
    flex: 1,
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
  addToListButton: {
    marginLeft: 8,
    padding: 10,
    backgroundColor: colors.primary + '15',
    borderRadius: 8,
  },
  errorText: {
    fontSize: 12,
    color: colors.danger,
    marginTop: 4,
  },
  floatingFooter: {
    backgroundColor: colors.card,
    padding: 16,
    borderTopWidth: 1,
    borderTopColor: colors.border,
  },
});
