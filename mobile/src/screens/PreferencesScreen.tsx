import React from 'react';
import { Text, StyleSheet, ScrollView, TouchableOpacity, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useNavigation, CommonActions } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { Ionicons } from '@expo/vector-icons';
import { Card } from '../components/common';
import { colors } from '../theme/colors';
import { useAppStatus, useWhatsAppStatus, useGCalStatus } from '../hooks';
import type { MainStackParamList } from '../navigation/MainNavigator';

type NavigationProp = NativeStackNavigationProp<MainStackParamList>;

interface PreferenceCardProps {
  title: string;
  description: string;
  icon: keyof typeof Ionicons.glyphMap;
  connected: boolean;
  onPress: () => void;
}

function PreferenceCard({ title, description, icon, connected, onPress }: PreferenceCardProps) {
  return (
    <TouchableOpacity onPress={onPress} activeOpacity={0.7}>
      <Card style={styles.preferenceCard}>
        <View style={styles.cardContent}>
          <View style={[styles.iconContainer, connected && styles.iconContainerConnected]}>
            <Ionicons
              name={icon}
              size={24}
              color={connected ? colors.primary : colors.textSecondary}
            />
          </View>
          <View style={styles.cardText}>
            <Text style={styles.cardTitle}>{title}</Text>
            <Text style={styles.cardDescription}>{description}</Text>
          </View>
          <Ionicons name="chevron-forward" size={20} color={colors.textSecondary} />
        </View>
      </Card>
    </TouchableOpacity>
  );
}

export function PreferencesScreen() {
  const navigation = useNavigation<NavigationProp>();
  const { data: appStatus } = useAppStatus();
  const { data: waStatus } = useWhatsAppStatus();
  const { data: gcalStatus } = useGCalStatus();

  const whatsappEnabled = appStatus?.whatsapp?.enabled ?? false;
  const gmailEnabled = appStatus?.gmail?.enabled ?? false;
  const whatsappConnected = waStatus?.connected ?? false;
  const gmailConnected = gcalStatus?.connected ?? false; // Gmail uses same Google OAuth

  // Navigate to home when tapping header (handled by parent)
  const handleGoHome = () => {
    navigation.dispatch(
      CommonActions.navigate({
        name: 'Home',
      })
    );
  };

  return (
    <SafeAreaView style={styles.container} edges={['top']}>
      {/* Header with Home navigation */}
      <TouchableOpacity style={styles.header} onPress={handleGoHome} activeOpacity={0.7}>
        <Text style={styles.headerTitle}>Alfred</Text>
      </TouchableOpacity>

      <ScrollView style={styles.scrollView} contentContainerStyle={styles.content}>
        <Text style={styles.sectionTitle}>Input Sources</Text>
        <Text style={styles.sectionDescription}>
          Configure which contacts and senders Alfred should scan for events.
        </Text>

        {whatsappEnabled && (
          <PreferenceCard
            title="WhatsApp"
            description="Manage tracked contacts and groups"
            icon="chatbubble-outline"
            connected={whatsappConnected}
            onPress={() => navigation.navigate('WhatsAppPreferences')}
          />
        )}

        {gmailEnabled && (
          <PreferenceCard
            title="Gmail"
            description="Manage tracked senders and domains"
            icon="mail-outline"
            connected={gmailConnected}
            onPress={() => navigation.navigate('GmailPreferences')}
          />
        )}

        {!whatsappEnabled && !gmailEnabled && (
          <Card style={styles.emptyCard}>
            <Text style={styles.emptyText}>
              No input sources configured.{'\n'}
              Complete onboarding to get started.
            </Text>
          </Card>
        )}
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.background,
  },
  header: {
    paddingHorizontal: 16,
    paddingTop: 16,
    paddingBottom: 8,
  },
  headerTitle: {
    fontSize: 24,
    fontWeight: 'bold',
    color: colors.primary,
  },
  scrollView: {
    flex: 1,
  },
  content: {
    padding: 16,
    paddingTop: 8,
  },
  sectionTitle: {
    fontSize: 20,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 8,
  },
  sectionDescription: {
    fontSize: 14,
    color: colors.textSecondary,
    marginBottom: 24,
    lineHeight: 20,
  },
  preferenceCard: {
    marginBottom: 12,
  },
  cardContent: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  iconContainer: {
    width: 48,
    height: 48,
    borderRadius: 24,
    backgroundColor: colors.background,
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 16,
  },
  iconContainerConnected: {
    backgroundColor: `${colors.primary}15`,
  },
  cardText: {
    flex: 1,
  },
  cardTitle: {
    fontSize: 17,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 4,
  },
  cardDescription: {
    fontSize: 13,
    color: colors.textSecondary,
  },
  emptyCard: {
    alignItems: 'center',
    paddingVertical: 32,
  },
  emptyText: {
    fontSize: 14,
    color: colors.textSecondary,
    textAlign: 'center',
    lineHeight: 20,
  },
});
