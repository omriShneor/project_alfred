import React, { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { Text, StyleSheet, ScrollView, TouchableOpacity, View, TextInput, Alert, KeyboardAvoidingView, Platform } from 'react-native';
import * as WebBrowser from 'expo-web-browser';
import * as Clipboard from 'expo-clipboard';
import * as ExpoLinking from 'expo-linking';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useNavigation, useFocusEffect, useRoute, type RouteProp } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { Ionicons } from '@expo/vector-icons';
import { Card, Button, LoadingSpinner } from '../components/common';
import { colors } from '../theme/colors';
import { API_BASE_URL } from '../config/api';
import {
  useWhatsAppStatus,
  useGCalStatus,
  useGCalSettings,
  useDisconnectWhatsApp,
  useGeneratePairingCode,
  useTelegramStatus,
  useSendTelegramCode,
  useVerifyTelegramCode,
  useDisconnectTelegram,
  useGmailStatus,
} from '../hooks';
import { useAppStatus } from '../hooks/useAppStatus';
import { disconnectGScope, requestAdditionalScopes, exchangeAddScopesCode } from '../api';
import type { MainStackParamList, TabParamList } from '../navigation/MainNavigator';

type NavigationProp = NativeStackNavigationProp<MainStackParamList>;
type PreferencesRouteProp = RouteProp<TabParamList, 'Preferences'>;

interface PreferenceCardProps {
  title: string;
  description: string;
  icon: keyof typeof Ionicons.glyphMap;
  connected: boolean;
  onPress: () => void;
}

type AccountIssueKey = 'whatsapp' | 'telegram' | 'gmail' | 'gcal';
type HeroStatusTone = 'warning' | 'neutral' | 'success';

interface AccountSummary {
  id: AccountIssueKey;
  label: string;
  connected: boolean;
  issue?: string;
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
  const route = useRoute<PreferencesRouteProp>();
  const { data: appStatus } = useAppStatus();
  const { data: waStatus, isLoading: waLoading, refetch: refetchWaStatus } = useWhatsAppStatus();
  const { data: gcalStatus, isLoading: gcalLoading, refetch: refetchGcalStatus } = useGCalStatus();
  const { data: gmailStatus, isLoading: gmailLoading, refetch: refetchGmailStatus } = useGmailStatus();
  const { data: gcalSettings } = useGCalSettings();
  const disconnectWhatsApp = useDisconnectWhatsApp();
  const generatePairingCode = useGeneratePairingCode();

  const [disconnectingGmail, setDisconnectingGmail] = useState(false);
  const [disconnectingGCal, setDisconnectingGCal] = useState(false);
  const [showWhatsAppConnect, setShowWhatsAppConnect] = useState(false);
  const [phoneNumber, setPhoneNumber] = useState('');
  const [pairingCode, setPairingCode] = useState<string | null>(null);
  const [showCopied, setShowCopied] = useState(false);

  // Telegram state
  const { data: telegramStatus, isLoading: telegramLoading, refetch: refetchTelegramStatus } = useTelegramStatus();
  const sendTelegramCode = useSendTelegramCode();
  const verifyTelegramCode = useVerifyTelegramCode();
  const disconnectTelegram = useDisconnectTelegram();
  const [showTelegramConnect, setShowTelegramConnect] = useState(false);
  const [telegramPhoneNumber, setTelegramPhoneNumber] = useState('');
  const [telegramCode, setTelegramCode] = useState('');
  const [telegramCodeSent, setTelegramCodeSent] = useState(false);
  const scrollViewRef = useRef<ScrollView>(null);
  const [accountsSectionY, setAccountsSectionY] = useState(0);


  // Check if any query is doing its initial load (no cached data yet)
  const isInitialLoading = (waLoading && !waStatus) || (gcalLoading && !gcalStatus) || (telegramLoading && !telegramStatus) || (gmailLoading && !gmailStatus);

  const whatsappConnected = waStatus?.connected ?? false;
  const telegramConnected = telegramStatus?.connected ?? false;
  // Check if Gmail has scopes (not just connected)
  const gmailHasScopes = gmailStatus?.has_scopes ?? false;
  // Check if Google Calendar has scopes (not just connected)
  const gcalHasScopes = gcalStatus?.has_scopes ?? false;

  const reauthSourceSet = useMemo(() => {
    const requested = route.params?.reauthSources ?? [];
    return new Set(requested);
  }, [route.params?.reauthSources]);

  const accountIssues = useMemo<Partial<Record<AccountIssueKey, string>>>(() => {
    const issues: Partial<Record<AccountIssueKey, string>> = {};

    const whatsappNeedsReauth =
      !whatsappConnected &&
      (Boolean(appStatus?.whatsapp?.enabled) || reauthSourceSet.has('whatsapp'));
    if (whatsappNeedsReauth) {
      issues.whatsapp =
        waStatus?.message ??
        'Session is no longer authenticated. Reconnect to resume tracking messages.';
    }

    const telegramNeedsReauth =
      !telegramConnected &&
      (Boolean(appStatus?.telegram?.enabled) || reauthSourceSet.has('telegram'));
    if (telegramNeedsReauth) {
      issues.telegram =
        telegramStatus?.message ??
        'Session is no longer authenticated. Reconnect to resume tracking messages.';
    }

    const gmailNeedsReauth =
      !gmailHasScopes &&
      (Boolean(appStatus?.gmail?.enabled) || reauthSourceSet.has('gmail'));
    if (gmailNeedsReauth) {
      issues.gmail =
        gmailStatus?.message ??
        'Gmail authorization is missing. Reconnect Gmail to keep scanning emails.';
    }

    const gcalNeedsReauth =
      !gcalHasScopes &&
      (Boolean(appStatus?.google_calendar?.enabled) ||
        reauthSourceSet.has('google_calendar'));
    if (gcalNeedsReauth) {
      issues.gcal =
        gcalStatus?.message ??
        'Google Calendar authorization is missing. Reconnect to restore calendar sync.';
    }

    return issues;
  }, [
    appStatus?.whatsapp?.enabled,
    appStatus?.telegram?.enabled,
    appStatus?.gmail?.enabled,
    appStatus?.google_calendar?.enabled,
    whatsappConnected,
    telegramConnected,
    gmailHasScopes,
    gcalHasScopes,
    waStatus?.message,
    telegramStatus?.message,
    gmailStatus?.message,
    gcalStatus?.message,
    reauthSourceSet,
  ]);

  const issueCount = Object.keys(accountIssues).length;

  const accountSummaries = useMemo<AccountSummary[]>(
    () => [
      {
        id: 'whatsapp',
        label: 'WhatsApp',
        connected: whatsappConnected,
        issue: accountIssues.whatsapp,
      },
      {
        id: 'telegram',
        label: 'Telegram',
        connected: telegramConnected,
        issue: accountIssues.telegram,
      },
      {
        id: 'gmail',
        label: 'Gmail',
        connected: gmailHasScopes,
        issue: accountIssues.gmail,
      },
      {
        id: 'gcal',
        label: 'Google Calendar',
        connected: gcalHasScopes,
        issue: accountIssues.gcal,
      },
    ],
    [
      whatsappConnected,
      telegramConnected,
      gmailHasScopes,
      gcalHasScopes,
      accountIssues.whatsapp,
      accountIssues.telegram,
      accountIssues.gmail,
      accountIssues.gcal,
    ]
  );

  const sortedAccounts = useMemo(
    () =>
      [...accountSummaries].sort((a, b) => {
        const aNeedsAttention = Boolean(a.issue);
        const bNeedsAttention = Boolean(b.issue);

        if (aNeedsAttention !== bNeedsAttention) {
          return aNeedsAttention ? -1 : 1;
        }

        if (a.connected !== b.connected) {
          return a.connected ? 1 : -1;
        }

        return 0;
      }),
    [accountSummaries]
  );

  const totalAccounts = accountSummaries.length;
  const connectedAccountsCount = accountSummaries.filter((account) => account.connected).length;
  const disconnectedAccountsCount = totalAccounts - connectedAccountsCount;
  const firstIssueAccount = accountSummaries.find((account) => Boolean(account.issue));
  const firstDisconnectedAccount = accountSummaries.find((account) => !account.connected);

  const primaryCtaLabel = firstIssueAccount
    ? `Reconnect ${firstIssueAccount.label}`
    : firstDisconnectedAccount
      ? `Connect ${firstDisconnectedAccount.label}`
      : 'Manage app connections';

  const heroDescription = firstIssueAccount
    ? 'Reconnect the highlighted app to restore full event and reminder coverage.'
    : firstDisconnectedAccount
      ? 'Connect another app so Alfred can catch more tasks, reminders, and events.'
      : 'All apps are connected. Review your connected app settings below.';

  const heroStatusTone: HeroStatusTone = firstIssueAccount
    ? 'warning'
    : firstDisconnectedAccount
      ? 'neutral'
      : 'success';

  const heroStatusText = firstIssueAccount
    ? `${issueCount} need${issueCount === 1 ? 's' : ''} attention`
    : disconnectedAccountsCount > 0
      ? `${disconnectedAccountsCount} to connect`
      : 'All good';

  // Reset WhatsApp connect UI when connected
  useEffect(() => {
    if (waStatus?.connected) {
      setShowWhatsAppConnect(false);
      setPairingCode(null);
      setPhoneNumber('');
    }
  }, [waStatus?.connected]);

  // Reset Telegram connect UI when connected
  useEffect(() => {
    if (telegramStatus?.connected) {
      setShowTelegramConnect(false);
      setTelegramCodeSent(false);
      setTelegramPhoneNumber('');
      setTelegramCode('');
    }
  }, [telegramStatus?.connected]);

  // Reset pairing states when navigating away from this screen
  useFocusEffect(
    useCallback(() => {
      return () => {
        // Called when screen loses focus - reset pairing states
        setShowWhatsAppConnect(false);
        setPairingCode(null);
        setPhoneNumber('');
        setShowTelegramConnect(false);
        setTelegramCodeSent(false);
        setTelegramPhoneNumber('');
        setTelegramCode('');
      };
    }, [])
  );

  const handleDisconnectWhatsApp = () => {
    Alert.alert(
      'Disconnect WhatsApp',
      'Are you sure you want to disconnect WhatsApp?',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Disconnect',
          style: 'destructive',
          onPress: async () => {
            try {
              await disconnectWhatsApp.mutateAsync();
              refetchWaStatus();
            } catch (error) {
              Alert.alert('Error', 'Failed to disconnect WhatsApp');
            }
          },
        },
      ]
    );
  };

  const handleDisconnectGmail = () => {
    Alert.alert(
      'Disconnect Gmail',
      'This will revoke Gmail access. You can reconnect anytime.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Disconnect',
          style: 'destructive',
          onPress: async () => {
            setDisconnectingGmail(true);
            try {
              await disconnectGScope('gmail');
              refetchGmailStatus();
            } catch (error) {
              Alert.alert('Error', 'Failed to disconnect Gmail');
            }
            setDisconnectingGmail(false);
          },
        },
      ]
    );
  };

  const handleDisconnectGCal = () => {
    Alert.alert(
      'Disconnect Google Calendar',
      'This will revoke Calendar access. You can reconnect anytime.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Disconnect',
          style: 'destructive',
          onPress: async () => {
            setDisconnectingGCal(true);
            try {
              await disconnectGScope('calendar');
              refetchGcalStatus();
            } catch (error) {
              Alert.alert('Error', 'Failed to disconnect Google Calendar');
            }
            setDisconnectingGCal(false);
          },
        },
      ]
    );
  };

  const handleConnectGmail = async () => {
    try {
      const backendCallbackUri = `${API_BASE_URL}/api/auth/callback`;
      const appDeepLink = ExpoLinking.createURL('oauth/callback');
      const response = await requestAdditionalScopes(['gmail'], backendCallbackUri);
      const result = await WebBrowser.openAuthSessionAsync(response.auth_url, appDeepLink);

      if (result.type === 'success' && result.url) {
        const parsed = ExpoLinking.parse(result.url);
        const code = parsed.queryParams?.code as string | undefined;
        if (code) {
          await exchangeAddScopesCode(code, ['gmail'], backendCallbackUri);
          refetchGmailStatus();
        }
      }
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to connect Gmail');
    }
  };

  const handleConnectGCal = async () => {
    try {
      const backendCallbackUri = `${API_BASE_URL}/api/auth/callback`;
      const appDeepLink = ExpoLinking.createURL('oauth/callback');
      const response = await requestAdditionalScopes(['calendar'], backendCallbackUri);
      const result = await WebBrowser.openAuthSessionAsync(response.auth_url, appDeepLink);

      if (result.type === 'success' && result.url) {
        const parsed = ExpoLinking.parse(result.url);
        const code = parsed.queryParams?.code as string | undefined;
        if (code) {
          await exchangeAddScopesCode(code, ['calendar'], backendCallbackUri);
          refetchGcalStatus();
        }
      }
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to connect Google Calendar');
    }
  };

  const handleConnectWhatsApp = async () => {
    if (!phoneNumber.trim()) {
      Alert.alert('Error', 'Please enter your phone number');
      return;
    }
    try {
      const result = await generatePairingCode.mutateAsync(phoneNumber.trim());
      setPairingCode(result.code);
    } catch (error: any) {
      Alert.alert('Error', error.response?.data?.error || 'Failed to generate pairing code');
    }
  };

  const handleShowWhatsAppConnect = () => {
    setShowWhatsAppConnect(true);
    setPairingCode(null);
    setPhoneNumber('');
  };

  const handleCopyCode = async () => {
    if (pairingCode) {
      await Clipboard.setStringAsync(pairingCode);
      setShowCopied(true);
      setTimeout(() => setShowCopied(false), 2000);
    }
  };

  // Telegram handlers
  const handleDisconnectTelegram = () => {
    Alert.alert(
      'Disconnect Telegram',
      'Are you sure you want to disconnect Telegram?',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Disconnect',
          style: 'destructive',
          onPress: async () => {
            try {
              await disconnectTelegram.mutateAsync();
              refetchTelegramStatus();
            } catch (error) {
              Alert.alert('Error', 'Failed to disconnect Telegram');
            }
          },
        },
      ]
    );
  };

  const handleShowTelegramConnect = () => {
    setShowTelegramConnect(true);
    setTelegramCodeSent(false);
    setTelegramPhoneNumber('');
    setTelegramCode('');
  };

  const handleSendTelegramCode = async () => {
    if (!telegramPhoneNumber.trim()) {
      Alert.alert('Error', 'Please enter your phone number');
      return;
    }
    try {
      await sendTelegramCode.mutateAsync(telegramPhoneNumber.trim());
      setTelegramCodeSent(true);
    } catch (error: any) {
      Alert.alert('Error', error.response?.data?.error || 'Failed to send verification code');
    }
  };

  const handleVerifyTelegramCode = async () => {
    if (!telegramCode.trim()) {
      Alert.alert('Error', 'Please enter the verification code');
      return;
    }
    try {
      await verifyTelegramCode.mutateAsync(telegramCode.trim());
      refetchTelegramStatus();
    } catch (error: any) {
      Alert.alert('Error', error.response?.data?.error || 'Failed to verify code');
    }
  };

  const scrollToAccountsSection = useCallback(() => {
    scrollViewRef.current?.scrollTo({
      y: Math.max(accountsSectionY - 12, 0),
      animated: true,
    });
  }, [accountsSectionY]);

  const getAccountStatusText = (connected: boolean, issue?: string) => {
    if (issue) {
      return 'Needs reconnection';
    }

    return connected ? 'Connected' : 'Not connected';
  };

  const handleOpenConnectionFlow = (accountId: AccountIssueKey) => {
    if (accountId === 'whatsapp') {
      handleShowWhatsAppConnect();
      setShowTelegramConnect(false);
      setTelegramCodeSent(false);
      setTelegramPhoneNumber('');
      setTelegramCode('');
      scrollToAccountsSection();
      return;
    }

    if (accountId === 'telegram') {
      handleShowTelegramConnect();
      setShowWhatsAppConnect(false);
      setPairingCode(null);
      setPhoneNumber('');
      scrollToAccountsSection();
      return;
    }

    if (accountId === 'gmail') {
      void handleConnectGmail();
      return;
    }

    void handleConnectGCal();
  };

  const handlePrimaryConnectionAction = () => {
    const targetAccount = firstIssueAccount ?? firstDisconnectedAccount;

    if (!targetAccount) {
      scrollToAccountsSection();
      return;
    }

    handleOpenConnectionFlow(targetAccount.id);
  };

  // Show loading state during initial data fetch to prevent flash
  if (isInitialLoading) {
    return (
      <SafeAreaView style={styles.container} edges={['left', 'right']}>
        <View style={styles.loadingContainer}>
          <LoadingSpinner />
        </View>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={styles.container} edges={['left', 'right']}>
      <KeyboardAvoidingView
        style={styles.keyboardAvoid}
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      >
        <ScrollView
          ref={scrollViewRef}
          style={styles.scrollView}
          contentContainerStyle={styles.content}
          keyboardShouldPersistTaps="handled"
        >
          <Card style={styles.heroCard}>
            <View style={styles.heroTopRow}>
              <View style={styles.heroTitleBlock}>
                <Text style={styles.heroEyebrow}>Connection Status</Text>
                <Text style={styles.heroTitle}>
                  {connectedAccountsCount} of {totalAccounts} apps connected
                </Text>
              </View>
              <View
                style={[
                  styles.heroStatusBadge,
                  heroStatusTone === 'warning'
                    ? styles.heroStatusBadgeWarning
                    : heroStatusTone === 'success'
                      ? styles.heroStatusBadgeSuccess
                      : styles.heroStatusBadgeNeutral,
                ]}
              >
                <Text
                  style={[
                    styles.heroStatusText,
                    heroStatusTone === 'warning'
                      ? styles.heroStatusTextWarning
                      : heroStatusTone === 'success'
                        ? styles.heroStatusTextSuccess
                        : styles.heroStatusTextNeutral,
                  ]}
                >
                  {heroStatusText}
                </Text>
              </View>
            </View>
            <View style={styles.progressTrack}>
              <View
                style={[
                  styles.progressFill,
                  { width: `${(connectedAccountsCount / totalAccounts) * 100}%` },
                ]}
              />
            </View>
            <Text style={styles.heroDescription}>{heroDescription}</Text>
            <Button
              title={primaryCtaLabel}
              onPress={handlePrimaryConnectionAction}
              style={styles.heroCtaButton}
            />
          </Card>

          <Text style={styles.sectionLabel}>Connected App Settings</Text>
          <Text style={styles.sectionDescription}>
            After an app is connected, choose what Alfred should monitor.
          </Text>

          {whatsappConnected && (
            <PreferenceCard
              title="WhatsApp"
              description="Manage chats Alfred scans for reminders and events"
              icon="chatbubble-outline"
              connected={whatsappConnected}
              onPress={() => navigation.navigate('WhatsAppPreferences')}
            />
          )}

          {telegramConnected && (
            <PreferenceCard
              title="Telegram"
              description="Manage chats Alfred scans for reminders and events"
              icon="paper-plane-outline"
              connected={telegramConnected}
              onPress={() => navigation.navigate('TelegramPreferences')}
            />
          )}

          {gmailHasScopes && (
            <PreferenceCard
              title="Gmail"
              description="Manage senders and domains Alfred monitors"
              icon="mail-outline"
              connected={gmailHasScopes}
              onPress={() => navigation.navigate('GmailPreferences')}
            />
          )}

          {gcalHasScopes && (
            <PreferenceCard
              title="Google Calendar"
              description={
                gcalSettings?.sync_enabled
                  ? `Syncing to ${gcalSettings.selected_calendar_name}`
                  : 'Choose which calendar Alfred syncs with'
              }
              icon="calendar-outline"
              connected={gcalHasScopes}
              onPress={() => navigation.navigate('GoogleCalendarPreferences')}
            />
          )}

          {!whatsappConnected && !telegramConnected && !gmailHasScopes && !gcalHasScopes && (
            <Card style={styles.emptyCard}>
              <Ionicons
                name="link-outline"
                size={26}
                color={colors.textMuted}
                style={styles.emptyIcon}
              />
              <Text style={styles.emptyText}>No connected apps yet</Text>
              <Text style={styles.emptySubtext}>
                Connect your first app below to start receiving suggestions.
              </Text>
              <Button
                title={primaryCtaLabel}
                onPress={handlePrimaryConnectionAction}
                style={styles.emptyCardButton}
              />
            </Card>
          )}

          <View
            onLayout={(event) => setAccountsSectionY(event.nativeEvent.layout.y)}
            style={styles.accountsSectionHeader}
          >
            <Text style={styles.sectionLabel}>App Connections</Text>
            <Text style={styles.sectionDescription}>
              Connect, reconnect, or disconnect the apps Alfred can use.
            </Text>
          </View>
          <Card>
            {sortedAccounts.map((account, index) => {
              const needsBorder = index > 0;

              if (account.id === 'whatsapp') {
                return (
                  <View key="whatsapp" style={needsBorder ? styles.accountRowBorder : undefined}>
                    <View style={styles.accountRow}>
                      <TouchableOpacity
                        style={styles.accountInfo}
                        onPress={() => showWhatsAppConnect && setShowWhatsAppConnect(false)}
                        activeOpacity={showWhatsAppConnect ? 0.7 : 1}
                      >
                        <Ionicons name="chatbubble-outline" size={20} color={colors.text} />
                        <View style={styles.accountText}>
                          <Text style={styles.accountName}>WhatsApp</Text>
                          <Text
                            style={[
                              styles.accountStatus,
                              accountIssues.whatsapp
                                ? styles.accountStatusWarning
                                : whatsappConnected
                                  ? styles.accountStatusConnected
                                  : undefined,
                            ]}
                          >
                            {getAccountStatusText(whatsappConnected, accountIssues.whatsapp)}
                          </Text>
                          {accountIssues.whatsapp ? (
                            <View style={styles.accountIssueRow}>
                              <Ionicons
                                name="warning-outline"
                                size={14}
                                color={colors.warning}
                              />
                              <Text style={styles.accountIssueText}>
                                {accountIssues.whatsapp}
                              </Text>
                            </View>
                          ) : null}
                        </View>
                      </TouchableOpacity>
                      {whatsappConnected ? (
                        <Button
                          title="Disconnect"
                          variant="outline"
                          onPress={handleDisconnectWhatsApp}
                          loading={disconnectWhatsApp.isPending}
                          style={styles.disconnectButton}
                        />
                      ) : !showWhatsAppConnect ? (
                        <Button
                          title={accountIssues.whatsapp ? 'Reconnect' : 'Connect'}
                          onPress={() => handleOpenConnectionFlow('whatsapp')}
                          style={styles.connectButton}
                        />
                      ) : null}
                    </View>
                    {!whatsappConnected && showWhatsAppConnect && (
                      <View style={styles.whatsappConnectSection}>
                        {!pairingCode ? (
                          <>
                            <Text style={styles.connectLabel}>
                              Enter your phone number (include country code)
                            </Text>
                            <TextInput
                              style={styles.input}
                              value={phoneNumber}
                              onChangeText={setPhoneNumber}
                              placeholder="+1234567890"
                              placeholderTextColor={colors.textSecondary}
                              keyboardType="phone-pad"
                              autoCapitalize="none"
                              autoCorrect={false}
                            />
                            <Button
                              title="Generate Pairing Code"
                              onPress={handleConnectWhatsApp}
                              loading={generatePairingCode.isPending}
                              style={styles.generateButton}
                            />
                          </>
                        ) : (
                          <>
                            <View style={styles.pairingCodeContainer}>
                              <Text style={styles.pairingCodeLabel}>Your pairing code</Text>
                              <View style={styles.pairingCodeRow}>
                                <Text style={styles.pairingCode}>{pairingCode}</Text>
                                <TouchableOpacity style={styles.copyButton} onPress={handleCopyCode}>
                                  <Ionicons
                                    name={showCopied ? 'checkmark' : 'copy-outline'}
                                    size={20}
                                    color={showCopied ? colors.success : colors.primary}
                                  />
                                </TouchableOpacity>
                              </View>
                            </View>
                            <Text style={styles.pairingInstructions}>
                              In WhatsApp: Settings → Linked Devices → Link with phone number
                            </Text>
                            <Button
                              title="Generate New Code"
                              onPress={handleConnectWhatsApp}
                              variant="outline"
                              loading={generatePairingCode.isPending}
                              style={styles.generateButton}
                            />
                          </>
                        )}
                      </View>
                    )}
                  </View>
                );
              }

              if (account.id === 'telegram') {
                return (
                  <View key="telegram" style={needsBorder ? styles.accountRowBorder : undefined}>
                    <View style={styles.accountRow}>
                      <TouchableOpacity
                        style={styles.accountInfo}
                        onPress={() => showTelegramConnect && setShowTelegramConnect(false)}
                        activeOpacity={showTelegramConnect ? 0.7 : 1}
                      >
                        <Ionicons name="paper-plane-outline" size={20} color={colors.text} />
                        <View style={styles.accountText}>
                          <Text style={styles.accountName}>Telegram</Text>
                          <Text
                            style={[
                              styles.accountStatus,
                              accountIssues.telegram
                                ? styles.accountStatusWarning
                                : telegramConnected
                                  ? styles.accountStatusConnected
                                  : undefined,
                            ]}
                          >
                            {getAccountStatusText(telegramConnected, accountIssues.telegram)}
                          </Text>
                          {accountIssues.telegram ? (
                            <View style={styles.accountIssueRow}>
                              <Ionicons
                                name="warning-outline"
                                size={14}
                                color={colors.warning}
                              />
                              <Text style={styles.accountIssueText}>
                                {accountIssues.telegram}
                              </Text>
                            </View>
                          ) : null}
                        </View>
                      </TouchableOpacity>
                      {telegramConnected ? (
                        <Button
                          title="Disconnect"
                          variant="outline"
                          onPress={handleDisconnectTelegram}
                          loading={disconnectTelegram.isPending}
                          style={styles.disconnectButton}
                        />
                      ) : !showTelegramConnect ? (
                        <Button
                          title={accountIssues.telegram ? 'Reconnect' : 'Connect'}
                          onPress={() => handleOpenConnectionFlow('telegram')}
                          style={styles.connectButton}
                        />
                      ) : null}
                    </View>
                    {!telegramConnected && showTelegramConnect && (
                      <View style={styles.telegramConnectSection}>
                        {!telegramCodeSent ? (
                          <>
                            <Text style={styles.connectLabel}>
                              Enter your phone number (include country code)
                            </Text>
                            <TextInput
                              style={styles.input}
                              value={telegramPhoneNumber}
                              onChangeText={setTelegramPhoneNumber}
                              placeholder="+1234567890"
                              placeholderTextColor={colors.textSecondary}
                              keyboardType="phone-pad"
                              autoCapitalize="none"
                              autoCorrect={false}
                            />
                            <Button
                              title="Send Verification Code"
                              onPress={handleSendTelegramCode}
                              loading={sendTelegramCode.isPending}
                              style={styles.generateButton}
                            />
                          </>
                        ) : (
                          <>
                            <Text style={styles.connectLabel}>
                              Enter the code sent to your Telegram app
                            </Text>
                            <TextInput
                              style={styles.input}
                              value={telegramCode}
                              onChangeText={setTelegramCode}
                              placeholder="12345"
                              placeholderTextColor={colors.textSecondary}
                              keyboardType="number-pad"
                              autoCapitalize="none"
                              autoCorrect={false}
                            />
                            <Button
                              title="Verify Code"
                              onPress={handleVerifyTelegramCode}
                              loading={verifyTelegramCode.isPending}
                              style={styles.generateButton}
                            />
                            <Button
                              title="Resend Code"
                              variant="outline"
                              onPress={handleSendTelegramCode}
                              loading={sendTelegramCode.isPending}
                              style={styles.generateButton}
                            />
                          </>
                        )}
                      </View>
                    )}
                  </View>
                );
              }

              if (account.id === 'gmail') {
                return (
                  <View key="gmail" style={needsBorder ? styles.accountRowBorder : undefined}>
                    <View style={styles.accountRow}>
                      <View style={styles.accountInfo}>
                        <Ionicons name="mail-outline" size={20} color={colors.text} />
                        <View style={styles.accountText}>
                          <Text style={styles.accountName}>Gmail</Text>
                          <Text
                            style={[
                              styles.accountStatus,
                              accountIssues.gmail
                                ? styles.accountStatusWarning
                                : gmailHasScopes
                                  ? styles.accountStatusConnected
                                  : undefined,
                            ]}
                          >
                            {getAccountStatusText(gmailHasScopes, accountIssues.gmail)}
                          </Text>
                          {accountIssues.gmail ? (
                            <View style={styles.accountIssueRow}>
                              <Ionicons
                                name="warning-outline"
                                size={14}
                                color={colors.warning}
                              />
                              <Text style={styles.accountIssueText}>
                                {accountIssues.gmail}
                              </Text>
                            </View>
                          ) : null}
                        </View>
                      </View>
                      {gmailHasScopes ? (
                        <Button
                          title="Disconnect"
                          variant="outline"
                          onPress={handleDisconnectGmail}
                          loading={disconnectingGmail}
                          style={styles.disconnectButton}
                        />
                      ) : (
                        <Button
                          title={accountIssues.gmail ? 'Reconnect' : 'Connect'}
                          onPress={() => handleOpenConnectionFlow('gmail')}
                          style={styles.connectButton}
                        />
                      )}
                    </View>
                  </View>
                );
              }

              if (account.id === 'gcal') {
                return (
                  <View key="gcal" style={needsBorder ? styles.accountRowBorder : undefined}>
                    <View style={styles.accountRow}>
                      <View style={styles.accountInfo}>
                        <Ionicons name="calendar-outline" size={20} color={colors.text} />
                        <View style={styles.accountText}>
                          <Text style={styles.accountName}>Google Calendar</Text>
                          <Text
                            style={[
                              styles.accountStatus,
                              accountIssues.gcal
                                ? styles.accountStatusWarning
                                : gcalHasScopes
                                  ? styles.accountStatusConnected
                                  : undefined,
                            ]}
                          >
                            {getAccountStatusText(gcalHasScopes, accountIssues.gcal)}
                          </Text>
                          {accountIssues.gcal ? (
                            <View style={styles.accountIssueRow}>
                              <Ionicons
                                name="warning-outline"
                                size={14}
                                color={colors.warning}
                              />
                              <Text style={styles.accountIssueText}>
                                {accountIssues.gcal}
                              </Text>
                            </View>
                          ) : null}
                        </View>
                      </View>
                      {gcalHasScopes ? (
                        <Button
                          title="Disconnect"
                          variant="outline"
                          onPress={handleDisconnectGCal}
                          loading={disconnectingGCal}
                          style={styles.disconnectButton}
                        />
                      ) : (
                        <Button
                          title={accountIssues.gcal ? 'Reconnect' : 'Connect'}
                          onPress={() => handleOpenConnectionFlow('gcal')}
                          style={styles.connectButton}
                        />
                      )}
                    </View>
                  </View>
                );
              }

              return null;
            })}
          </Card>

        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.background,
  },
  keyboardAvoid: {
    flex: 1,
  },
  loadingContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  scrollView: {
    flex: 1,
  },
  content: {
    padding: 16,
    paddingTop: 16,
    paddingBottom: 32,
  },
  heroCard: {
    borderRadius: 14,
    borderWidth: 1,
    borderColor: colors.primary + '22',
    backgroundColor: colors.infoBackground,
    paddingVertical: 14,
    paddingHorizontal: 14,
    marginBottom: 14,
  },
  heroTopRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    justifyContent: 'space-between',
    marginBottom: 10,
  },
  heroTitleBlock: {
    flex: 1,
    marginRight: 12,
  },
  heroEyebrow: {
    fontSize: 12,
    fontWeight: '700',
    color: colors.primary,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
    marginBottom: 5,
  },
  heroTitle: {
    fontSize: 21,
    fontWeight: '700',
    color: colors.text,
    lineHeight: 28,
  },
  heroStatusBadge: {
    paddingHorizontal: 10,
    paddingVertical: 6,
    borderRadius: 999,
    borderWidth: 1,
  },
  heroStatusBadgeWarning: {
    borderColor: colors.warning + '55',
    backgroundColor: colors.warning + '12',
  },
  heroStatusBadgeNeutral: {
    borderColor: colors.border,
    backgroundColor: colors.background,
  },
  heroStatusBadgeSuccess: {
    borderColor: colors.success + '55',
    backgroundColor: colors.success + '12',
  },
  heroStatusText: {
    fontSize: 12,
    fontWeight: '700',
  },
  heroStatusTextWarning: {
    color: colors.warning,
  },
  heroStatusTextNeutral: {
    color: colors.textSecondary,
  },
  heroStatusTextSuccess: {
    color: colors.success,
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
  heroDescription: {
    marginTop: 10,
    marginBottom: 12,
    fontSize: 13,
    color: colors.textSecondary,
    lineHeight: 18,
  },
  heroCtaButton: {
    alignSelf: 'flex-start',
  },
  sectionLabel: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.textSecondary,
    marginTop: 8,
    marginBottom: 8,
    marginLeft: 4,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  sectionDescription: {
    fontSize: 14,
    color: colors.textSecondary,
    marginBottom: 16,
    lineHeight: 20,
  },
  accountsSectionHeader: {
    marginTop: 6,
  },
  // Account styles
  accountRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 12,
  },
  accountRowBorder: {
    borderTopWidth: 1,
    borderTopColor: colors.border,
  },
  accountInfo: {
    flexDirection: 'row',
    alignItems: 'center',
    flex: 1,
  },
  accountText: {
    marginLeft: 12,
    flex: 1,
  },
  accountName: {
    fontSize: 16,
    fontWeight: '500',
    color: colors.text,
  },
  accountStatus: {
    fontSize: 13,
    color: colors.textSecondary,
    marginTop: 2,
  },
  accountStatusConnected: {
    color: colors.success,
  },
  accountStatusWarning: {
    color: colors.warning,
  },
  accountIssueRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    marginTop: 4,
    marginRight: 8,
  },
  accountIssueText: {
    flex: 1,
    marginLeft: 5,
    fontSize: 12,
    lineHeight: 16,
    color: colors.warning,
  },
  disconnectButton: {
    paddingHorizontal: 12,
    paddingVertical: 6,
  },
  connectButton: {
    paddingHorizontal: 16,
    paddingVertical: 6,
  },
  whatsappConnectSection: {
    marginTop: 12,
    paddingBottom: 16,
  },
  telegramConnectSection: {
    marginTop: 12,
    paddingBottom: 16,
  },
  connectLabel: {
    fontSize: 13,
    color: colors.textSecondary,
    marginBottom: 8,
  },
  input: {
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 8,
    padding: 12,
    fontSize: 16,
    color: colors.text,
    backgroundColor: colors.background,
  },
  generateButton: {
    marginTop: 12,
  },
  pairingCodeContainer: {
    alignItems: 'center',
    backgroundColor: colors.background,
    borderRadius: 8,
    padding: 16,
    marginBottom: 12,
  },
  pairingCodeLabel: {
    fontSize: 12,
    color: colors.textSecondary,
    marginBottom: 4,
  },
  pairingCodeRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    position: 'relative',
    width: '100%',
  },
  pairingCode: {
    fontSize: 28,
    fontWeight: '700',
    color: colors.primary,
    letterSpacing: 4,
  },
  copyButton: {
    padding: 8,
    position: 'absolute',
    right: 0,
  },
  pairingInstructions: {
    fontSize: 13,
    color: colors.textSecondary,
    textAlign: 'center',
    marginBottom: 12,
  },
  // Preference card styles
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
  emptyIcon: {
    marginBottom: 16,
  },
  emptyText: {
    fontSize: 16,
    fontWeight: '600',
    color: colors.text,
    textAlign: 'center',
    marginBottom: 8,
  },
  emptySubtext: {
    fontSize: 14,
    color: colors.textSecondary,
    textAlign: 'center',
    lineHeight: 20,
    marginBottom: 20,
    paddingHorizontal: 16,
  },
  emptyCardButton: {
    minWidth: 200,
  },
});
