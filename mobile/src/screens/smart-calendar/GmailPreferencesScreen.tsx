import React, { useMemo, useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TouchableOpacity,
  FlatList,
  Alert,
} from 'react-native';
import { Feather } from '@expo/vector-icons';
import * as WebBrowser from 'expo-web-browser';
import * as ExpoLinking from 'expo-linking';
import { useQueryClient } from '@tanstack/react-query';
import { LoadingSpinner, Card, Button } from '../../components/common';
import { AddSourceModal } from '../../components/sources/AddSourceModal';
import { colors } from '../../theme/colors';
import {
  useGmailStatus,
  useEmailSources,
  useCreateEmailSource,
  useDeleteEmailSource,
  useTopContacts,
  useAddCustomSource,
  useSearchGmailContacts,
  useDebounce,
} from '../../hooks';
import { requestAdditionalScopes, exchangeAddScopesCode } from '../../api/auth';
import { API_BASE_URL } from '../../config/api';
import type { EmailSource, EmailSourceType, TopContact } from '../../types/gmail';
import type { SourceTopContact } from '../../types/channel';

export function GmailPreferencesScreen() {
  const queryClient = useQueryClient();
  const { data: gmailStatus, isLoading: statusLoading } = useGmailStatus();
  const [addSourceModalVisible, setAddSourceModalVisible] = useState(false);
  const [isConnecting, setIsConnecting] = useState(false);

  const { data: sources, isLoading: sourcesLoading } = useEmailSources();
  const { data: topContacts, isLoading: contactsLoading } = useTopContacts({
    enabled: Boolean(gmailStatus?.connected && gmailStatus?.has_scopes && addSourceModalVisible),
  });

  const [searchQuery, setSearchQuery] = useState('');
  const debouncedQuery = useDebounce(searchQuery, 300);
  const { data: gmailSearchResults, isLoading: searchLoading } = useSearchGmailContacts(debouncedQuery);

  const createSource = useCreateEmailSource();
  const deleteSource = useDeleteEmailSource();
  const addCustomSource = useAddCustomSource();

  const trackedSenderIds = useMemo(() => {
    const map = new Map<string, number>();
    for (const source of sources || []) {
      if (source.type !== 'sender' || !source.enabled) {
        continue;
      }
      const normalized = source.identifier.trim().toLowerCase();
      if (normalized) {
        map.set(normalized, source.id);
      }
    }
    return map;
  }, [sources]);

  // Handle connecting Gmail (requesting Gmail scope)
  const handleConnectGmail = async () => {
    setIsConnecting(true);
    try {
      // Use the backend callback URL as the redirect (same pattern as login)
      const backendCallbackUri = `${API_BASE_URL}/api/auth/callback`;
      const appDeepLink = ExpoLinking.createURL('oauth/callback');

      // Request Gmail scope
      const response = await requestAdditionalScopes(['gmail'], backendCallbackUri);

      // Open browser for authorization
      const result = await WebBrowser.openAuthSessionAsync(response.auth_url, appDeepLink);

      if (result.type === 'success' && result.url) {
        // Extract code from callback URL
        const parsed = ExpoLinking.parse(result.url);
        const code = parsed.queryParams?.code as string | undefined;

        if (code) {
          // Exchange code and add Gmail scopes
          await exchangeAddScopesCode(code, ['gmail'], backendCallbackUri);
          // Refresh Gmail status
          queryClient.invalidateQueries({ queryKey: ['gmailStatus'] });
          Alert.alert('Success', 'Gmail access authorized successfully!');
        } else {
          throw new Error('No authorization code received');
        }
      }
    } catch (error: any) {
      console.error('Failed to connect Gmail:', error);
      Alert.alert('Error', error.message || 'Failed to connect Gmail');
    } finally {
      setIsConnecting(false);
    }
  };

  // Map Gmail TopContact to SourceTopContact for the shared modal
  const mappedContacts: SourceTopContact[] = (topContacts || []).map((contact: TopContact) => {
    const normalizedEmail = contact.email.trim().toLowerCase();
    return {
      identifier: normalizedEmail,
      name: contact.name || contact.email,
      secondary_label: contact.email,
      message_count: contact.email_count,
      is_tracked: trackedSenderIds.has(normalizedEmail),
      channel_id: trackedSenderIds.get(normalizedEmail),
      type: 'sender' as const,
    };
  });

  // Map Gmail search results to SourceTopContact format
  const mappedSearchResults: SourceTopContact[] | undefined = gmailSearchResults?.map((contact: TopContact) => {
    const normalizedEmail = contact.email.trim().toLowerCase();
    return {
      identifier: normalizedEmail,
      name: contact.name || contact.email,
      secondary_label: contact.email,
      message_count: contact.email_count,
      is_tracked: trackedSenderIds.has(normalizedEmail),
      channel_id: trackedSenderIds.get(normalizedEmail),
      type: 'sender' as const,
    };
  });

  const handleDeleteSource = (source: EmailSource) => {
    Alert.alert(
      'Remove Sender',
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
              Alert.alert('Error', error.message || 'Failed to remove sender');
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
    setAddSourceModalVisible(true);
  };

  // Validation for custom email/domain input
  const validateCustomInput = (value: string): string | null => {
    if (!value.trim()) return null;
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    const domainRegex = /^@?[a-zA-Z0-9][a-zA-Z0-9.-]*\.[a-zA-Z]{2,}$/;
    if (emailRegex.test(value) || domainRegex.test(value)) {
      return null;
    }
    return 'Enter a valid email (user@domain.com) or domain (domain.com)';
  };

  // Handler for adding selected contacts from the modal
  const handleAddContacts = async (contacts: SourceTopContact[]) => {
    for (const contact of contacts) {
      await createSource.mutateAsync({
        type: 'sender',
        identifier: contact.identifier,
        name: contact.name,
      });
    }
  };

  // Handler for adding custom email/domain
  const handleAddCustom = async (value: string) => {
    await addCustomSource.mutateAsync({ value });
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

  const renderSourceItem = ({ item }: { item: EmailSource }) => (
    <View style={styles.sourceItem}>
      <View style={styles.sourceInfo}>
        <View style={styles.sourceHeader}>
          <Text
            style={styles.sourceName}
            numberOfLines={1}
            ellipsizeMode="tail"
          >
            {item.name}
          </Text>
          <View style={[styles.sourceTypeBadge, { backgroundColor: getSourceTypeColor(item.type) + '20' }]}>
            <Text style={[styles.sourceTypeText, { color: getSourceTypeColor(item.type) }]}>
              {getSourceTypeLabel(item.type)}
            </Text>
          </View>
        </View>
        <Text
          style={styles.sourceIdentifier}
          numberOfLines={1}
          ellipsizeMode="tail"
        >
          {item.identifier}
        </Text>
      </View>
      <View style={styles.sourceActions}>
        <TouchableOpacity style={styles.deleteButton} onPress={() => handleDeleteSource(item)}>
          <Feather name="trash-2" size={18} color={colors.danger} />
        </TouchableOpacity>
      </View>
    </View>
  );

  // Show connect UI if Gmail scopes not granted
  if (!statusLoading && gmailStatus && !gmailStatus.has_scopes) {
    return (
      <View style={styles.screen}>
        <ScrollView style={styles.container} contentContainerStyle={styles.content}>
          <Card>
            <View style={styles.connectContainer}>
              <View style={styles.connectIconContainer}>
                <Feather name="mail" size={48} color={colors.primary} />
              </View>
              <Text style={styles.connectTitle}>Connect Gmail</Text>
              <Text style={styles.connectDescription}>
                Grant Gmail access so Alfred can detect events, reminders, and tasks from selected senders.
                Alfred only reads email content and never sends, modifies, or deletes messages.
              </Text>
              <Button
                title="Connect Gmail"
                onPress={handleConnectGmail}
                loading={isConnecting}
                style={styles.connectButton}
              />
            </View>
          </Card>
        </ScrollView>
      </View>
    );
  }

  return (
    <View style={styles.screen}>
      <ScrollView style={styles.container} contentContainerStyle={styles.content}>
        <View style={styles.sectionHeader}>
          <Text style={styles.sectionTitle}>Gmail Senders</Text>
          <TouchableOpacity
            style={styles.addButton}
            onPress={handleOpenAddSourceModal}
          >
            <Feather name="plus" size={18} color={colors.primary} />
            <Text style={styles.addButtonText}>Add Sender</Text>
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
              <Text style={styles.emptyStateText}>No Gmail senders selected</Text>
              <Text style={styles.emptyStateSubtext}>
                Add senders or domains to track for events, reminders, and tasks
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
        title="Add Gmail Sender"
        topContacts={mappedContacts}
        contactsLoading={contactsLoading}
        searchResults={mappedSearchResults}
        searchLoading={searchLoading}
        onSearchQueryChange={setSearchQuery}
        customInputPlaceholder="e.g. boss@work.com"
        customInputKeyboardType="email-address"
        validateCustomInput={validateCustomInput}
        onAddContacts={handleAddContacts}
        onAddCustom={handleAddCustom}
        addContactsLoading={createSource.isPending}
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
  sourceItem: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 12,
  },
  sourceInfo: {
    flex: 1,
    minWidth: 0,
    marginRight: 12,
  },
  sourceHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 4,
    minWidth: 0,
  },
  sourceName: {
    flexShrink: 1,
    minWidth: 0,
    fontSize: 15,
    fontWeight: '500',
    color: colors.text,
    marginRight: 8,
  },
  sourceTypeBadge: {
    flexShrink: 0,
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
    flexShrink: 0,
  },
  deleteButton: {
    marginLeft: 0,
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
  // Connect Gmail styles
  connectContainer: {
    alignItems: 'center',
    paddingVertical: 32,
    paddingHorizontal: 24,
  },
  connectIconContainer: {
    width: 80,
    height: 80,
    borderRadius: 40,
    backgroundColor: colors.primary + '15',
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 16,
  },
  connectTitle: {
    fontSize: 20,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 8,
  },
  connectDescription: {
    fontSize: 14,
    color: colors.textSecondary,
    textAlign: 'center',
    lineHeight: 20,
    marginBottom: 24,
  },
  connectButton: {
    minWidth: 200,
  },
});
