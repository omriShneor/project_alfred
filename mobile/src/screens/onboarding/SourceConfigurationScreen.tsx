import React, { useCallback } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  Alert,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRoute, useNavigation, useFocusEffect } from '@react-navigation/native';
import type { RouteProp } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { Ionicons } from '@expo/vector-icons';
import { Button, Card, LoadingSpinner } from '../../components/common';
import { colors } from '../../theme/colors';
import { useChannels, useCompleteOnboarding } from '../../hooks';
import { useTelegramChannels } from '../../hooks/useTelegram';
import { useEmailSources } from '../../hooks/useGmail';
import type { OnboardingParamList } from '../../navigation/OnboardingNavigator';

type RouteProps = RouteProp<OnboardingParamList, 'SourceConfiguration'>;
type NavigationProp = NativeStackNavigationProp<OnboardingParamList, 'SourceConfiguration'>;

interface ServiceCardProps {
  icon: keyof typeof Ionicons.glyphMap;
  title: string;
  description: string;
  configured: boolean;
  onPress: () => void;
}

function ServiceCard({ icon, title, description, configured, onPress }: ServiceCardProps) {
  return (
    <Card style={[styles.card, configured && styles.cardConfigured]}>
      <View style={styles.serviceHeader}>
        <View style={styles.serviceInfo}>
          <Ionicons name={icon} size={22} color={colors.text} />
          <View style={styles.serviceText}>
            <Text style={styles.serviceTitle}>{title}</Text>
            <Text style={styles.serviceDescription}>{description}</Text>
          </View>
        </View>
        <View style={styles.serviceStatus}>
          <View style={[styles.statusDot, { backgroundColor: configured ? colors.success : colors.textSecondary }]} />
          <Text style={styles.statusLabel}>{configured ? 'Ready' : 'Needs setup'}</Text>
        </View>
      </View>
      <Button
        title={configured ? 'Manage' : 'Choose'}
        onPress={onPress}
        variant={configured ? 'outline' : 'primary'}
        style={styles.configureButton}
      />
    </Card>
  );
}

export function SourceConfigurationScreen() {
  const route = useRoute<RouteProps>();
  const navigation = useNavigation<NavigationProp>();
  const { whatsappEnabled, telegramEnabled, gmailEnabled } = route.params;

  // Fetch channels and sources
  const { data: whatsappChannels, refetch: refetchWhatsAppChannels, isLoading: whatsappLoading } = useChannels();
  const { data: telegramChannels, refetch: refetchTelegramChannels, isLoading: telegramLoading } = useTelegramChannels();
  const { data: emailSources, refetch: refetchEmailSources, isLoading: emailLoading } = useEmailSources();
  const completeOnboarding = useCompleteOnboarding();

  // Derive configuration status
  const whatsappConfigured = (whatsappChannels?.some((channel) => channel.enabled) ?? false);
  const telegramConfigured = (telegramChannels?.some((channel) => channel.enabled) ?? false);
  const gmailConfigured = (emailSources?.some((source) => source.enabled) ?? false);

  const requiredAppsCount = [whatsappEnabled, telegramEnabled, gmailEnabled].filter(Boolean).length;
  const configuredAppsCount = [
    whatsappEnabled ? whatsappConfigured : false,
    telegramEnabled ? telegramConfigured : false,
    gmailEnabled ? gmailConfigured : false,
  ].filter(Boolean).length;
  const hasConfiguredSource = configuredAppsCount > 0;
  const finishButtonTitle = hasConfiguredSource
    ? 'Finish setup'
    : 'Select at least one contact or sender';

  // Refetch sources when screen gains focus (user returns from preference screens)
  useFocusEffect(
    useCallback(() => {
      refetchWhatsAppChannels();
      refetchTelegramChannels();
      refetchEmailSources();
    }, [refetchWhatsAppChannels, refetchTelegramChannels, refetchEmailSources])
  );

  const handleConfigureWhatsApp = () => {
    navigation.navigate('WhatsAppPreferences');
  };

  const handleConfigureTelegram = () => {
    navigation.navigate('TelegramPreferences');
  };

  const handleConfigureGmail = () => {
    navigation.navigate('GmailPreferences');
  };

  const handleContinue = async () => {
    try {
      await completeOnboarding.mutateAsync({
        whatsapp_enabled: whatsappEnabled,
        telegram_enabled: telegramEnabled,
        gmail_enabled: gmailEnabled,
      });
      // RootNavigator will automatically switch to MainNavigator
      // when onboarding_complete becomes true (query is invalidated in the hook)
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to complete onboarding');
    }
  };

  // Show loading state during initial data fetch
  if ((whatsappLoading && !whatsappChannels) || (telegramLoading && !telegramChannels) || (emailLoading && !emailSources)) {
    return (
      <SafeAreaView style={styles.safeArea} edges={['top']}>
        <View style={styles.loadingContainer}>
          <LoadingSpinner />
        </View>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={styles.safeArea} edges={['top']}>
      <ScrollView
        style={styles.container}
        contentContainerStyle={styles.content}
      >
        <Card style={styles.heroCard}>
          <View style={styles.heroTopRow}>
            <Text style={styles.step}>Step 3 of 3</Text>
            <View
              style={[
                styles.heroStatusBadge,
                hasConfiguredSource ? styles.heroStatusBadgeSuccess : styles.heroStatusBadgeWarning,
              ]}
            >
              <Text
                style={[
                  styles.heroStatusText,
                  hasConfiguredSource ? styles.heroStatusTextSuccess : styles.heroStatusTextWarning,
                ]}
              >
                {configuredAppsCount}/{requiredAppsCount} ready
              </Text>
            </View>
          </View>
          <Text style={styles.title}>Choose What Alfred Should Monitor</Text>
          <Text style={styles.description}>
            Pick contacts, chats, or senders from your connected apps. You need at least one to finish setup.
          </Text>
          <View style={styles.progressTrack}>
            <View
              style={[
                styles.progressFill,
                { width: `${(configuredAppsCount / Math.max(requiredAppsCount, 1)) * 100}%` },
              ]}
            />
          </View>
        </Card>

        {whatsappEnabled && (
          <ServiceCard
            icon="chatbubble-outline"
            title="WhatsApp"
            description="Choose chats Alfred should scan"
            configured={whatsappConfigured}
            onPress={handleConfigureWhatsApp}
          />
        )}

        {telegramEnabled && (
          <ServiceCard
            icon="paper-plane-outline"
            title="Telegram"
            description="Choose chats Alfred should scan"
            configured={telegramConfigured}
            onPress={handleConfigureTelegram}
          />
        )}

        {gmailEnabled && (
          <ServiceCard
            icon="mail-outline"
            title="Gmail"
            description="Choose senders and domains Alfred should scan"
            configured={gmailConfigured}
            onPress={handleConfigureGmail}
          />
        )}

        {!hasConfiguredSource && (
          <View style={styles.statusSummary}>
            <Ionicons name="information-circle-outline" size={16} color={colors.textSecondary} />
            <Text style={styles.statusSummaryText}>
              Configure at least one contact or sender to continue
            </Text>
          </View>
        )}

        <Button
          title={finishButtonTitle}
          onPress={handleContinue}
          disabled={!hasConfiguredSource}
          loading={completeOnboarding.isPending}
          style={styles.continueButton}
        />
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: colors.background,
  },
  container: {
    flex: 1,
  },
  content: {
    padding: 24,
    paddingBottom: 48,
  },
  loadingContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  heroCard: {
    borderRadius: 14,
    borderWidth: 1,
    borderColor: colors.primary + '22',
    backgroundColor: colors.infoBackground,
    marginBottom: 16,
  },
  heroTopRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 10,
  },
  step: {
    fontSize: 12,
    color: colors.primary,
    fontWeight: '700',
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  heroStatusBadge: {
    borderRadius: 999,
    borderWidth: 1,
    paddingHorizontal: 10,
    paddingVertical: 6,
  },
  heroStatusBadgeSuccess: {
    borderColor: colors.success + '45',
    backgroundColor: colors.success + '12',
  },
  heroStatusBadgeWarning: {
    borderColor: colors.warning + '45',
    backgroundColor: colors.warning + '12',
  },
  heroStatusText: {
    fontSize: 12,
    fontWeight: '700',
  },
  heroStatusTextSuccess: {
    color: colors.success,
  },
  heroStatusTextWarning: {
    color: colors.warning,
  },
  title: {
    fontSize: 27,
    fontWeight: '700',
    color: colors.text,
    marginBottom: 8,
  },
  description: {
    fontSize: 14,
    color: colors.textSecondary,
    lineHeight: 20,
    marginBottom: 12,
  },
  progressTrack: {
    height: 8,
    borderRadius: 999,
    backgroundColor: colors.border,
    overflow: 'hidden',
  },
  progressFill: {
    height: '100%',
    backgroundColor: colors.primary,
  },
  card: {
    marginBottom: 16,
  },
  cardConfigured: {
    borderWidth: 1,
    borderColor: colors.success + '28',
  },
  serviceHeader: {
    marginBottom: 12,
  },
  serviceInfo: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    marginBottom: 12,
  },
  serviceText: {
    marginLeft: 12,
    flex: 1,
    marginTop: 1,
  },
  serviceTitle: {
    fontSize: 17,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 2,
  },
  serviceDescription: {
    fontSize: 13,
    color: colors.textSecondary,
    lineHeight: 18,
    marginTop: 4,
  },
  serviceStatus: {
    flexDirection: 'row',
    alignItems: 'center',
    marginTop: 8,
  },
  statusDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    marginRight: 6,
  },
  statusLabel: {
    fontSize: 13,
    color: colors.textSecondary,
    fontWeight: '500',
  },
  configureButton: {
    marginTop: 0,
  },
  statusSummary: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: 2,
    marginBottom: 16,
    paddingHorizontal: 16,
  },
  statusSummaryText: {
    fontSize: 13,
    color: colors.textSecondary,
    marginLeft: 8,
  },
  continueButton: {
    marginTop: 8,
  },
});
