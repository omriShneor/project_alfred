import React, { useState, useEffect } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  TextInput,
  KeyboardTypeOptions,
  Keyboard,
  Platform,
} from 'react-native';
import { Feather } from '@expo/vector-icons';
import { Modal, Button, LoadingSpinner } from '../common';
import { colors } from '../../theme/colors';
import type { SourceTopContact } from '../../types/channel';

interface CustomEntry {
  identifier: string;
  name: string;
}

export interface AddSourceModalProps {
  visible: boolean;
  onClose: () => void;
  title: string;
  // Top contacts
  topContacts: SourceTopContact[] | undefined;
  contactsLoading: boolean;
  // Custom input
  customInputPlaceholder: string;
  customInputKeyboardType: KeyboardTypeOptions;
  validateCustomInput: (value: string) => string | null;
  // Actions - always uses 'primary' (Alfred Calendar) as the target
  onAddContacts: (contacts: SourceTopContact[]) => Promise<void>;
  onAddCustom: (value: string) => Promise<void>;
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
  customInputPlaceholder,
  customInputKeyboardType,
  validateCustomInput,
  onAddContacts,
  onAddCustom,
  addContactsLoading,
  addCustomLoading,
}: AddSourceModalProps) {
  const [selectedContacts, setSelectedContacts] = useState<Set<string>>(new Set());
  const [customEntries, setCustomEntries] = useState<CustomEntry[]>([]);
  const [customInput, setCustomInput] = useState('');
  const [customInputError, setCustomInputError] = useState<string | null>(null);
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

  // Reset state when modal opens
  React.useEffect(() => {
    if (visible) {
      setSelectedContacts(new Set());
      setCustomEntries([]);
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

  const removeCustomEntry = (identifier: string) => {
    setCustomEntries((prev) => prev.filter((e) => e.identifier !== identifier));
    setSelectedContacts((prev) => {
      const newSet = new Set(prev);
      newSet.delete(identifier);
      return newSet;
    });
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
    if (topContacts?.some((c) => c.identifier === trimmedValue)) {
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
    setIsAdding(true);
    try {
      // Add selected top contacts (always uses Alfred Calendar / 'primary')
      const contactsToAdd = (topContacts || []).filter(
        (c) => selectedContacts.has(c.identifier) && !c.is_tracked
      );
      if (contactsToAdd.length > 0) {
        await onAddContacts(contactsToAdd);
      }

      // Add custom entries
      const customToAdd = customEntries.filter((e) => selectedContacts.has(e.identifier));
      for (const entry of customToAdd) {
        await onAddCustom(entry.identifier);
      }

      setSelectedContacts(new Set());
      setCustomEntries([]);
      onClose();
    } catch {
      // Error handled by parent
    } finally {
      setIsAdding(false);
    }
  };

  // Total selected count (from top contacts + custom entries)
  const totalSelected = selectedContacts.size;

  const renderContactItem = (item: SourceTopContact, index: number, showSeparator: boolean) => {
    const selected = selectedContacts.has(item.identifier);
    return (
      <React.Fragment key={item.identifier}>
        {showSeparator && <View style={styles.separator} />}
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
              {item.name || item.secondary_label || item.identifier}
            </Text>
            {item.name && item.secondary_label && (
              <Text style={styles.contactIdentifier} numberOfLines={1}>
                {item.secondary_label}
              </Text>
            )}
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

  return (
    <Modal
      visible={visible}
      onClose={onClose}
      title={title}
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
              title={`Add ${totalSelected} Contact${totalSelected !== 1 ? 's' : ''}`}
              onPress={handleAddAllSelected}
              loading={isAdding || addContactsLoading || addCustomLoading}
            />
          </View>
        ) : undefined
      }
    >
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
            {topContacts.map((item, index) => renderContactItem(item, index, index > 0))}
          </View>
        ) : (
          <View style={styles.emptyContacts}>
            <Text style={styles.emptyContactsText}>No suggested contacts yet</Text>
            <Text style={styles.emptyContactsSubtext}>
              Add a contact manually below
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
            placeholder={customInputPlaceholder}
            placeholderTextColor={colors.textSecondary}
            keyboardType={customInputKeyboardType}
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
  );
}

const styles = StyleSheet.create({
  sectionLabel: {
    fontSize: 14,
    fontWeight: '500',
    color: colors.text,
    marginBottom: 8,
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
  contactIdentifier: {
    fontSize: 13,
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
  customSection: {
    borderTopWidth: 1,
    borderTopColor: colors.border,
    paddingTop: 16,
    paddingBottom: 40,
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
    paddingBottom: 32,
    borderTopWidth: 1,
    borderTopColor: colors.border,
  },
});
