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
  configured: boolean;
  onPress: () => void;
}

function ServiceCard({ icon, title, configured, onPress }: ServiceCardProps) {
  return (
    <Card style={styles.card}>
      <View style={styles.serviceHeader}>
        <View style={styles.serviceInfo}>
          <Ionicons name={icon} size={24} color={colors.text} />
          <View style={styles.serviceText}>
            <Text style={styles.serviceTitle}>{title}</Text>
          </View>
        </View>
        <View style={styles.serviceStatus}>
          <View style={[styles.statusDot, { backgroundColor: configured ? colors.success : colors.textSecondary }]} />
          <Text style={styles.statusLabel}>{configured ? 'Configured' : 'Not configured'}</Text>
        </View>
      </View>
      <Button
        title={configured ? 'Manage' : 'Configure'}
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
  const whatsappConfigured = (whatsappChannels?.length ?? 0) > 0;
  const telegramConfigured = (telegramChannels?.length ?? 0) > 0;
  const gmailConfigured = (emailSources?.length ?? 0) > 0;

  // Check if at least one source is configured
  const hasConfiguredSource = whatsappConfigured || telegramConfigured || gmailConfigured;

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
        <Text style={styles.step}>Step 3 of 3</Text>
        <Text style={styles.title}>Select Contacts and Senders</Text>
        <Text style={styles.description}>
          Choose which contacts or email senders Alfred should use for event, reminder, and task suggestions.
        </Text>

        {whatsappEnabled && (
          <ServiceCard
            icon="chatbubble-outline"
            title="WhatsApp"
            configured={whatsappConfigured}
            onPress={handleConfigureWhatsApp}
          />
        )}

        {telegramEnabled && (
          <ServiceCard
            icon="paper-plane-outline"
            title="Telegram"
            configured={telegramConfigured}
            onPress={handleConfigureTelegram}
          />
        )}

        {gmailEnabled && (
          <ServiceCard
            icon="mail-outline"
            title="Gmail"
            configured={gmailConfigured}
            onPress={handleConfigureGmail}
          />
        )}

        {!hasConfiguredSource && (
          <View style={styles.statusSummary}>
            <Ionicons name="information-circle-outline" size={16} color={colors.textSecondary} />
            <Text style={styles.statusSummaryText}>
              Configure at least one source to continue
            </Text>
          </View>
        )}

        <Button
          title="Continue"
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
  step: {
    fontSize: 14,
    color: colors.primary,
    fontWeight: '600',
    marginBottom: 8,
  },
  title: {
    fontSize: 28,
    fontWeight: 'bold',
    color: colors.text,
    marginBottom: 12,
  },
  description: {
    fontSize: 15,
    color: colors.textSecondary,
    lineHeight: 22,
    marginBottom: 32,
  },
  card: {
    marginBottom: 16,
  },
  serviceHeader: {
    marginBottom: 16,
  },
  serviceInfo: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    marginBottom: 12,
  },
  serviceText: {
    marginLeft: 12,
    flex: 1,
    marginTop: 3
  },
  serviceTitle: {
    fontSize: 17,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 2,
  },
  serviceDescription: {
    fontSize: 14,
    color: colors.textSecondary,
    lineHeight: 20,
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
    marginTop: 8,
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
