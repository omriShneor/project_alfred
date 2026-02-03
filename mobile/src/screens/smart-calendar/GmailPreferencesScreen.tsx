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
  ActivityIndicator,
} from 'react-native';
import { Feather } from '@expo/vector-icons';
import * as WebBrowser from 'expo-web-browser';
import * as ExpoLinking from 'expo-linking';
import { useQueryClient } from '@tanstack/react-query';
import { LoadingSpinner, Card } from '../../components/common';
import { AddSourceModal } from '../../components/sources/AddSourceModal';
import { colors } from '../../theme/colors';
import {
  useGmailStatus,
  useEmailSources,
  useCreateEmailSource,
  useUpdateEmailSource,
  useDeleteEmailSource,
  useTopContacts,
  useAddCustomSource,
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
  const { data: topContacts, isLoading: contactsLoading } = useTopContacts();

  const createSource = useCreateEmailSource();
  const updateSource = useUpdateEmailSource();
  const deleteSource = useDeleteEmailSource();
  const addCustomSource = useAddCustomSource();

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
  const mappedContacts: SourceTopContact[] = (topContacts || []).map((contact: TopContact) => ({
    identifier: contact.email,
    name: contact.name || contact.email,
    secondary_label: contact.email,
    message_count: contact.email_count,
    is_tracked: contact.is_tracked,
    channel_id: contact.source_id,
    type: 'sender' as const,
  }));

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
      if (!contact.is_tracked) {
        await createSource.mutateAsync({
          type: 'sender',
          identifier: contact.identifier,
          name: contact.name,
        });
      }
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
                Grant Gmail access to scan your emails for events and reminders.
                We only read emails - we never send, modify, or delete anything.
              </Text>
              <TouchableOpacity
                style={[styles.connectButton, isConnecting && styles.connectButtonDisabled]}
                onPress={handleConnectGmail}
                disabled={isConnecting}
              >
                {isConnecting ? (
                  <ActivityIndicator color="#fff" size="small" />
                ) : (
                  <>
                    <Feather name="link" size={18} color="#fff" />
                    <Text style={styles.connectButtonText}>Connect Gmail</Text>
                  </>
                )}
              </TouchableOpacity>
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

      <AddSourceModal
        visible={addSourceModalVisible}
        onClose={() => setAddSourceModalVisible(false)}
        title="Add Email Source"
        topContacts={mappedContacts}
        contactsLoading={contactsLoading}
        customInputPlaceholder="e.g. boss@work.com or acme.com"
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
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: colors.primary,
    paddingVertical: 14,
    paddingHorizontal: 28,
    borderRadius: 12,
    minWidth: 180,
  },
  connectButtonDisabled: {
    opacity: 0.7,
  },
  connectButtonText: {
    fontSize: 16,
    fontWeight: '600',
    color: '#fff',
    marginLeft: 8,
  },
});
