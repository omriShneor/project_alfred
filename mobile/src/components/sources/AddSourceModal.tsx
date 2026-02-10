import React, { useState, useEffect } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  TextInput,
  KeyboardTypeOptions,
  Keyboard,
} from 'react-native';
import { Feather } from '@expo/vector-icons';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Modal, Button, LoadingSpinner } from '../common';
import { colors } from '../../theme/colors';
import type { SourceTopContact } from '../../types/channel';

interface CustomEntry {
  identifier: string;
  name: string;
}

const emptyContacts: SourceTopContact[] = [];
const TOP_SUGGESTIONS_LIMIT = 8;

export interface AddSourceModalProps {
  visible: boolean;
  onClose: () => void;
  title: string;
  // Top contacts
  topContacts: SourceTopContact[] | undefined;
  contactsLoading: boolean;
  // Search
  searchResults?: SourceTopContact[];
  searchLoading?: boolean;
  onSearchQueryChange?: (query: string) => void;
  // Custom input
  customInputPlaceholder: string;
  customInputKeyboardType: KeyboardTypeOptions;
  validateCustomInput: (value: string) => string | null;
  blockManualAddWhenSearchResults?: boolean;
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
  searchResults,
  searchLoading,
  onSearchQueryChange,
  customInputPlaceholder,
  customInputKeyboardType,
  validateCustomInput,
  blockManualAddWhenSearchResults = false,
  onAddContacts,
  onAddCustom,
  addContactsLoading,
  addCustomLoading,
}: AddSourceModalProps) {
  const insets = useSafeAreaInsets();
  const [selectedContacts, setSelectedContacts] = useState<Set<string>>(new Set());
  const [customEntries, setCustomEntries] = useState<CustomEntry[]>([]);
  const [customInput, setCustomInput] = useState('');
  const [customInputError, setCustomInputError] = useState<string | null>(null);
  const [isAdding, setIsAdding] = useState(false);

  const normalizedQuery = customInput.trim();
  const isSearching = normalizedQuery.length >= 2;

  const entityLabel = React.useMemo(() => {
    const normalizedTitle = title.toLowerCase();
    if (normalizedTitle.includes('sender')) return 'sender';
    if (normalizedTitle.includes('chat')) return 'chat';
    if (normalizedTitle.includes('contact')) return 'contact';
    return 'entry';
  }, [title]);
  const entityLabelPlural = entityLabel === 'entry' ? 'entries' : `${entityLabel}s`;
  const entityLabelPluralTitle =
    entityLabelPlural.charAt(0).toUpperCase() + entityLabelPlural.slice(1);

  const topSuggestedContacts = React.useMemo(() => {
    const suggestions = [...(topContacts ?? emptyContacts)];
    suggestions.sort((a, b) => b.message_count - a.message_count);
    return suggestions.slice(0, TOP_SUGGESTIONS_LIMIT);
  }, [topContacts]);

  const displayedContacts = isSearching
    ? (searchResults ?? emptyContacts)
    : topSuggestedContacts;
  const displayedLoading = isSearching ? (searchLoading ?? false) : contactsLoading;

  const allKnownContacts = React.useMemo(() => {
    const byIdentifier = new Map<string, SourceTopContact>();
    for (const contact of topContacts ?? emptyContacts) {
      byIdentifier.set(contact.identifier, contact);
    }
    for (const contact of searchResults ?? emptyContacts) {
      byIdentifier.set(contact.identifier, contact);
    }
    return Array.from(byIdentifier.values());
  }, [topContacts, searchResults]);

  React.useEffect(() => {
    if (onSearchQueryChange) {
      onSearchQueryChange(normalizedQuery);
    }
  }, [normalizedQuery, onSearchQueryChange]);

  React.useEffect(() => {
    if (visible) {
      setSelectedContacts(new Set());
      setCustomEntries([]);
      setCustomInput('');
      setCustomInputError(null);
    }
  }, [visible]);

  useEffect(() => {
    setSelectedContacts((prev) => {
      if (prev.size === 0) {
        return prev;
      }

      const allowed = new Set<string>();
      for (const contact of allKnownContacts) {
        allowed.add(contact.identifier);
      }
      for (const entry of customEntries) {
        allowed.add(entry.identifier);
      }

      let changed = false;
      const next = new Set<string>();
      for (const id of prev) {
        if (allowed.has(id)) {
          next.add(id);
        } else {
          changed = true;
        }
      }

      return changed ? next : prev;
    });
  }, [allKnownContacts, customEntries]);

  const toggleContactSelection = (identifier: string) => {
    setSelectedContacts((prev) => {
      const next = new Set(prev);
      if (next.has(identifier)) {
        next.delete(identifier);
      } else {
        next.add(identifier);
      }
      return next;
    });
  };

  const removeCustomEntry = (identifier: string) => {
    setCustomEntries((prev) => prev.filter((entry) => entry.identifier !== identifier));
    setSelectedContacts((prev) => {
      const next = new Set(prev);
      next.delete(identifier);
      return next;
    });
  };

  const handleAddCustomToList = () => {
    const error = validateCustomInput(customInput);
    if (error) {
      setCustomInputError(error);
      return;
    }

    const trimmedValue = customInput.trim();
    const normalizedValue = trimmedValue.toLowerCase();

    if (customEntries.some((entry) => entry.identifier.toLowerCase() === normalizedValue)) {
      setCustomInputError('Already selected');
      return;
    }

    const matchingContacts = displayedContacts.filter((contact) => {
      const fields = [contact.name, contact.push_name, contact.secondary_label, contact.identifier]
        .filter((value): value is string => Boolean(value))
        .map((value) => value.toLowerCase());
      return fields.some((value) => value === normalizedValue);
    });

    if (matchingContacts.length > 1) {
      setCustomInputError('Multiple matches found above');
      return;
    }

    if (matchingContacts.length === 1) {
      const match = matchingContacts[0];
      if (match.is_tracked) {
        setCustomInputError('Already added');
        return;
      }

      setSelectedContacts((prev) => new Set(prev).add(match.identifier));
      setCustomInput('');
      setCustomInputError(null);
      Keyboard.dismiss();
      return;
    }

    if (blockManualAddWhenSearchResults && isSearching && displayedContacts.length > 0) {
      setCustomInputError('Select a result from the list above');
      return;
    }

    setCustomEntries((prev) => [...prev, { identifier: trimmedValue, name: trimmedValue }]);
    setSelectedContacts((prev) => new Set(prev).add(trimmedValue));
    setCustomInput('');
    setCustomInputError(null);
    Keyboard.dismiss();
  };

  const handleAddAllSelected = async () => {
    setIsAdding(true);
    try {
      const contactsToAdd = allKnownContacts.filter(
        (contact) => selectedContacts.has(contact.identifier) && !contact.is_tracked
      );
      if (contactsToAdd.length > 0) {
        await onAddContacts(contactsToAdd);
      }

      const customToAdd = customEntries.filter((entry) => selectedContacts.has(entry.identifier));
      for (const entry of customToAdd) {
        await onAddCustom(entry.identifier);
      }

      setSelectedContacts(new Set());
      setCustomEntries([]);
      onClose();
    } catch {
      // Error handled by parent screen
    } finally {
      setIsAdding(false);
    }
  };

  const handleClearSelection = () => {
    setSelectedContacts(new Set());
    setCustomEntries([]);
  };

  const totalSelected = selectedContacts.size;
  const primaryButtonText =
    totalSelected > 0
      ? `Add ${totalSelected} ${totalSelected === 1 ? entityLabel : entityLabelPlural}`
      : 'Add Selected';
  const selectionSummaryText =
    totalSelected > 0
      ? `${totalSelected} ${totalSelected === 1 ? entityLabel : entityLabelPlural} selected`
      : `Select ${entityLabelPlural} to add`;

  const renderContactItem = (
    item: SourceTopContact,
    showSeparator: boolean
  ) => {
    const selected = selectedContacts.has(item.identifier);
    const subtitle =
      item.push_name && item.push_name !== item.name ? item.push_name : item.secondary_label;

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
            {item.name && subtitle && (
              <Text style={styles.contactIdentifier} numberOfLines={1}>
                {subtitle}
              </Text>
            )}
          </View>
          <View style={styles.contactAction}>
            {item.is_tracked ? (
              <View style={styles.trackedBadge}>
                <Text style={styles.trackedBadgeText}>Tracked</Text>
              </View>
            ) : selected ? (
              <Feather name="check-square" size={20} color={colors.primary} />
            ) : (
              <Feather name="square" size={20} color={colors.textSecondary} />
            )}
          </View>
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
                <Text style={styles.customBadgeText}>Manual</Text>
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
              accessibilityRole="button"
              accessibilityLabel={`Remove ${entry.identifier}`}
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
        <View style={[styles.floatingFooter, { paddingBottom: 16 + insets.bottom }]}>
          <View style={styles.composerSection}>
            <Text style={styles.sectionLabel}>Search or add manually</Text>
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
                returnKeyType="done"
                onSubmitEditing={handleAddCustomToList}
                onBlur={() => {
                  if (customInput.trim()) {
                    setCustomInputError(validateCustomInput(customInput));
                  }
                }}
              />
            </View>
            {customInputError ? <Text style={styles.errorText}>{customInputError}</Text> : null}
          </View>

          <View style={styles.footerSummaryRow}>
            <Text style={styles.footerSummaryText}>{selectionSummaryText}</Text>
            {totalSelected > 0 && (
              <TouchableOpacity
                style={styles.clearButton}
                onPress={handleClearSelection}
                accessibilityRole="button"
                accessibilityLabel="Clear selection"
              >
                <Text style={styles.clearButtonText}>Clear</Text>
              </TouchableOpacity>
            )}
          </View>
          <Button
            title={primaryButtonText}
            onPress={handleAddAllSelected}
            loading={isAdding || addContactsLoading || addCustomLoading}
            disabled={totalSelected === 0}
          />
        </View>
      }
    >
      <View style={styles.guidanceCard}>
        <Feather
          name="info"
          size={14}
          color={colors.primary}
          style={styles.guidanceIcon}
        />
        <Text style={styles.guidanceText}>
          Select {entityLabelPlural} from suggestions or add one manually below.
        </Text>
      </View>

      {customEntries.length > 0 && (
        <View style={styles.customEntriesSection}>
          <Text style={styles.sectionLabel}>Added manually</Text>
          <View style={styles.contactList}>
            {customEntries.map((entry, index) => renderCustomEntry(entry, index))}
          </View>
        </View>
      )}

      <View style={styles.suggestedSection}>
        <Text style={styles.sectionLabel}>Suggested {entityLabelPluralTitle}</Text>
        {displayedLoading ? (
          <LoadingSpinner />
        ) : displayedContacts.length > 0 ? (
          <View style={styles.contactList}>
            {displayedContacts.map((item, index) => renderContactItem(item, index > 0))}
          </View>
        ) : (
          <View style={styles.emptyContacts}>
            <Text style={styles.emptyContactsText}>
              {normalizedQuery
                ? `No matching ${entityLabelPlural}`
                : `No suggested ${entityLabelPlural} yet`}
            </Text>
            <Text style={styles.emptyContactsSubtext}>
              Add a {entityLabel} manually below
            </Text>
          </View>
        )}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  guidanceCard: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    borderWidth: 1,
    borderColor: colors.primary + '28',
    backgroundColor: colors.infoBackground,
    borderRadius: 10,
    paddingVertical: 10,
    paddingHorizontal: 10,
    marginBottom: 14,
  },
  guidanceIcon: {
    marginTop: 1,
    marginRight: 8,
  },
  guidanceText: {
    flex: 1,
    fontSize: 13,
    color: colors.textSecondary,
    lineHeight: 18,
  },
  sectionLabel: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 8,
  },
  customEntriesSection: {
    marginBottom: 16,
  },
  suggestedSection: {
    marginBottom: 16,
  },
  contactList: {},
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
    opacity: 0.65,
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
  contactAction: {
    minWidth: 64,
    alignItems: 'flex-end',
  },
  trackedBadge: {
    backgroundColor: colors.success + '20',
    borderRadius: 999,
    paddingHorizontal: 8,
    paddingVertical: 3,
  },
  trackedBadgeText: {
    fontSize: 11,
    fontWeight: '600',
    color: colors.success,
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
  composerSection: {
    marginBottom: 10,
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
  errorText: {
    fontSize: 12,
    color: colors.danger,
    marginTop: 4,
  },
  floatingFooter: {
    backgroundColor: colors.background,
    paddingHorizontal: 16,
    paddingTop: 8,
    paddingBottom: 12,
    borderTopWidth: 1,
    borderTopColor: colors.border,
  },
  footerSummaryRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 8,
  },
  footerSummaryText: {
    fontSize: 13,
    color: colors.textSecondary,
    fontWeight: '500',
  },
  clearButton: {
    paddingVertical: 4,
    paddingHorizontal: 8,
    borderRadius: 999,
    backgroundColor: colors.background,
  },
  clearButtonText: {
    fontSize: 12,
    fontWeight: '600',
    color: colors.primary,
  },
});
