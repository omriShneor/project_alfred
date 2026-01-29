import React from 'react';
import { View, Text, StyleSheet, ScrollView, TouchableOpacity } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { Feather } from '@expo/vector-icons';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { Card } from '../../components/common';
import { colors } from '../../theme/colors';
import { useFeatures } from '../../hooks';
import type { SmartCalendarHubStackParamList } from '../../navigation/DrawerNavigator';

type NavigationProp = NativeStackNavigationProp<SmartCalendarHubStackParamList>;

interface IntegrationItemProps {
  icon: keyof typeof Feather.glyphMap;
  title: string;
  description: string;
  onPress: () => void;
}

function IntegrationItem({ icon, title, description, onPress }: IntegrationItemProps) {
  return (
    <TouchableOpacity onPress={onPress} activeOpacity={0.7}>
      <Card style={styles.integrationCard}>
        <View style={styles.integrationContent}>
          <View style={styles.iconContainer}>
            <Feather name={icon} size={24} color={colors.primary} />
          </View>
          <View style={styles.textContainer}>
            <Text style={styles.integrationTitle}>{title}</Text>
            <Text style={styles.integrationDescription}>{description}</Text>
          </View>
          <Feather name="chevron-right" size={20} color={colors.textSecondary} />
        </View>
      </Card>
    </TouchableOpacity>
  );
}

export function InputIntegrationsScreen() {
  const navigation = useNavigation<NavigationProp>();
  const { data: features } = useFeatures();

  const showWhatsApp = features?.smart_calendar?.inputs?.whatsapp?.enabled ?? false;
  const showGmail = features?.smart_calendar?.inputs?.email?.enabled ?? false;

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <Text style={styles.sectionLabel}>INPUT INTEGRATIONS</Text>

      {showWhatsApp && (
        <IntegrationItem
          icon="message-circle"
          title="WhatsApp Preferences"
          description="Choose which contacts and groups to scan for events"
          onPress={() => navigation.navigate('WhatsAppPreferences')}
        />
      )}

      {showGmail && (
        <IntegrationItem
          icon="mail"
          title="Gmail Preferences"
          description="Choose which email senders to scan for events"
          onPress={() => navigation.navigate('GmailPreferences')}
        />
      )}

      {!showWhatsApp && !showGmail && (
        <Card style={styles.emptyCard}>
          <View style={styles.emptyState}>
            <Feather name="inbox" size={40} color={colors.textSecondary} />
            <Text style={styles.emptyStateText}>No input integrations enabled</Text>
            <Text style={styles.emptyStateSubtext}>
              Enable integrations in Assistant Capabilities to get started
            </Text>
          </View>
        </Card>
      )}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.background,
  },
  content: {
    padding: 16,
  },
  sectionLabel: {
    fontSize: 12,
    fontWeight: '600',
    color: colors.textSecondary,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
    marginBottom: 12,
    marginLeft: 4,
  },
  integrationCard: {
    marginBottom: 12,
  },
  integrationContent: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  iconContainer: {
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: colors.primary + '15',
    alignItems: 'center',
    justifyContent: 'center',
    marginRight: 12,
  },
  textContainer: {
    flex: 1,
    marginRight: 8,
  },
  integrationTitle: {
    fontSize: 16,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 2,
  },
  integrationDescription: {
    fontSize: 13,
    color: colors.textSecondary,
  },
  emptyCard: {
    marginTop: 8,
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
