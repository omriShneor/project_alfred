import React, { useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  FlatList,
  TextInput,
  KeyboardTypeOptions,
} from 'react-native';
import { Feather } from '@expo/vector-icons';
import { Modal, Select, Button, LoadingSpinner } from '../common';
import { colors } from '../../theme/colors';
import type { SourceTopContact } from '../../types/channel';
import type { Calendar } from '../../types/event';

export interface AddSourceModalProps {
  visible: boolean;
  onClose: () => void;
  title: string;
  // Top contacts
  topContacts: SourceTopContact[] | undefined;
  contactsLoading: boolean;
  // Calendars
  calendars: Calendar[] | undefined;
  // Custom input
  customInputPlaceholder: string;
  customInputKeyboardType: KeyboardTypeOptions;
  validateCustomInput: (value: string) => string | null;
  // Actions
  onAddContacts: (contacts: SourceTopContact[], calendarId: string) => Promise<void>;
  onAddCustom: (value: string, calendarId: string) => Promise<void>;
  // Loading states
  addContactsLoading: boolean;
  addCustomLoading: boolean;
}

export function AddSourceModal({
  visible,
  onClose,
  title,
  topContacts,
  contactsLoading,
  calendars,
  customInputPlaceholder,
  customInputKeyboardType,
  validateCustomInput,
  onAddContacts,
  onAddCustom,
  addContactsLoading,
  addCustomLoading,
}: AddSourceModalProps) {
  const [selectedContacts, setSelectedContacts] = useState<Set<string>>(new Set());
  const [customInput, setCustomInput] = useState('');
  const [customInputError, setCustomInputError] = useState<string | null>(null);
  const [selectedCalendarId, setSelectedCalendarId] = useState<string>('');

  // Set default calendar when calendars load
  React.useEffect(() => {
    if (calendars && calendars.length > 0 && !selectedCalendarId) {
      const primaryCalendar = calendars.find((c) => c.primary);
      setSelectedCalendarId(primaryCalendar?.id || calendars[0].id);
    }
  }, [calendars, selectedCalendarId]);

  // Reset state when modal opens
  React.useEffect(() => {
    if (visible) {
      setSelectedContacts(new Set());
      setCustomInput('');
      setCustomInputError(null);
    }
  }, [visible]);

  const toggleContactSelection = (identifier: string) => {
    setSelectedContacts((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(identifier)) {
        newSet.delete(identifier);
      } else {
        newSet.add(identifier);
      }
      return newSet;
    });
  };

  const handleAddContacts = async () => {
    if (selectedContacts.size === 0 || !selectedCalendarId) return;

    const contactsToAdd = (topContacts || []).filter(
      (c) => selectedContacts.has(c.identifier) && !c.is_tracked
    );

    try {
      await onAddContacts(contactsToAdd, selectedCalendarId);
      setSelectedContacts(new Set());
      onClose();
    } catch {
      // Error handled by parent
    }
  };

  const handleAddCustom = async () => {
    const error = validateCustomInput(customInput);
    if (error) {
      setCustomInputError(error);
      return;
    }

    if (!selectedCalendarId) {
      setCustomInputError('Please select a calendar');
      return;
    }

    try {
      await onAddCustom(customInput.trim(), selectedCalendarId);
      setCustomInput('');
      setCustomInputError(null);
      onClose();
    } catch {
      // Error handled by parent
    }
  };

  const calendarOptions = calendars?.map((c) => ({
    label: c.summary + (c.primary ? ' (Primary)' : ''),
    value: c.id,
  })) || [];

  const availableContacts = topContacts?.filter((c) => !c.is_tracked) || [];
  const hasSelectableContacts = availableContacts.length > 0;

  const renderContactItem = ({ item }: { item: SourceTopContact }) => {
    const selected = selectedContacts.has(item.identifier);
    return (
      <TouchableOpacity
        style={[
          styles.contactItem,
          selected && styles.contactItemSelected,
          item.is_tracked && styles.contactItemDisabled,
        ]}
        onPress={() => !item.is_tracked && toggleContactSelection(item.identifier)}
        disabled={item.is_tracked}
      >
        <View style={styles.contactInfo}>
          <Text style={styles.contactName} numberOfLines={1}>
            {item.name || `+${item.identifier}`}
          </Text>
          {item.name && item.name !== item.identifier && (
            <Text style={styles.contactIdentifier} numberOfLines={1}>
              +{item.identifier}
            </Text>
          )}
          <Text style={styles.contactCount}>{item.message_count} messages</Text>
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

  return (
    <Modal visible={visible} onClose={onClose} title={title} scrollable={false}>
      {/* Calendar Selection */}
      <View style={styles.calendarSection}>
        <Text style={styles.sectionLabel}>Target Calendar</Text>
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
            keyExtractor={(item) => item.identifier}
            renderItem={renderContactItem}
            style={styles.contactList}
            ItemSeparatorComponent={() => <View style={styles.separator} />}
          />
        ) : (
          <View style={styles.emptyContacts}>
            <Text style={styles.emptyContactsText}>No suggested contacts yet</Text>
            <Text style={styles.emptyContactsSubtext}>
              Add a contact manually below
            </Text>
          </View>
        )}

        {hasSelectableContacts && selectedContacts.size > 0 && (
          <Button
            title={`Add ${selectedContacts.size} Contact${selectedContacts.size !== 1 ? 's' : ''}`}
            onPress={handleAddContacts}
            loading={addContactsLoading}
            style={styles.addContactsButton}
          />
        )}
      </View>

      {/* Custom Input Section */}
      <View style={styles.customSection}>
        <Text style={styles.sectionLabel}>Or add manually</Text>
        <TextInput
          style={[styles.customInput, customInputError && styles.customInputError]}
          value={customInput}
          onChangeText={(text) => {
            setCustomInput(text);
            if (customInputError) setCustomInputError(null);
          }}
          placeholder={customInputPlaceholder}
          placeholderTextColor={colors.textSecondary}
          keyboardType={customInputKeyboardType}
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
          loading={addCustomLoading}
          disabled={!customInput.trim()}
          style={styles.addCustomButton}
        />
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  calendarSection: {
    marginBottom: 16,
  },
  sectionLabel: {
    fontSize: 14,
    fontWeight: '500',
    color: colors.text,
    marginBottom: 8,
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
  contactIdentifier: {
    fontSize: 13,
    color: colors.textSecondary,
    marginTop: 2,
  },
  contactCount: {
    fontSize: 12,
    color: colors.textSecondary,
    marginTop: 2,
  },
  separator: {
    height: 1,
    backgroundColor: colors.border,
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
