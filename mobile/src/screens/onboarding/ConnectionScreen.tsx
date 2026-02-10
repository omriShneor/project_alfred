import React, { useState, useEffect, useCallback } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  Alert,
  TouchableOpacity,
  Image,
  TextInput,
  KeyboardAvoidingView,
  Platform,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRoute, useNavigation, useFocusEffect } from '@react-navigation/native';
import type { RouteProp } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import * as WebBrowser from 'expo-web-browser';
import * as Clipboard from 'expo-clipboard';
import * as Notifications from 'expo-notifications';
import { Feather } from '@expo/vector-icons';
import { Button, Card } from '../../components/common';
import { colors } from '../../theme/colors';
import { useQueryClient } from '@tanstack/react-query';
import {
  useWhatsAppStatus,
  useGCalStatus,
  useGmailStatus,
  useGeneratePairingCode,
  useTelegramStatus,
  useSendTelegramCode,
  useVerifyTelegramCode,
} from '../../hooks';
import { useRequestAdditionalScopes, useExchangeAddScopesCode } from '../../hooks/useIncrementalAuth';
import { ScopeType } from '../../api/auth';
import type { OnboardingParamList } from '../../navigation/OnboardingNavigator';

type RouteProps = RouteProp<OnboardingParamList, 'Connection'>;
type NavigationProp = NativeStackNavigationProp<OnboardingParamList, 'Connection'>;
type IntegrationStatusType = 'pending' | 'connecting' | 'needs_access' | 'available' | 'error';

function getStatusColor(status: IntegrationStatusType) {
  switch (status) {
    case 'available':
      return colors.success;
    case 'connecting':
      return colors.warning;
    case 'needs_access':
      return colors.warning;
    case 'error':
      return colors.danger;
    default:
      return colors.textSecondary;
  }
}

function getStatusLabel(status: IntegrationStatusType) {
  switch (status) {
    case 'available':
      return 'Connected';
    case 'connecting':
      return 'Connecting...';
    case 'needs_access':
      return 'Needs access';
    case 'error':
      return 'Error';
    default:
      return 'Not connected';
  }
}

function getGoogleScopeLabel(scope: ScopeType) {
  if (scope === 'gmail') {
    return 'Gmail';
  }
  if (scope === 'calendar') {
    return 'Google Calendar';
  }
  return 'Google';
}

function joinScopeLabels(labels: string[]) {
  if (labels.length <= 1) {
    return labels[0] || '';
  }
  if (labels.length === 2) {
    return `${labels[0]} and ${labels[1]}`;
  }
  return `${labels.slice(0, -1).join(', ')}, and ${labels[labels.length - 1]}`;
}

function GoogleSignInButton({
  onPress,
  loading,
  title,
  disabled,
}: {
  onPress: () => void;
  loading?: boolean;
  title: string;
  disabled?: boolean;
}) {
  return (
    <TouchableOpacity
      style={styles.googleButton}
      onPress={onPress}
      disabled={disabled || loading}
      activeOpacity={0.7}
    >
      <Image
        source={require('../../../assets/google-logo.png')}
        style={styles.googleLogo}
      />
      <Text style={styles.googleButtonText}>
        {loading ? 'Connecting...' : title}
      </Text>
    </TouchableOpacity>
  );
}

export function ConnectionScreen() {
  const route = useRoute<RouteProps>();
  const navigation = useNavigation<NavigationProp>();
  const queryClient = useQueryClient();

  const { whatsappEnabled, telegramEnabled, gmailEnabled, gcalEnabled } = route.params;

  // WhatsApp state
  const [phoneNumber, setPhoneNumber] = useState('');
  const [pairingCode, setPairingCode] = useState<string | null>(null);
  const [showCopied, setShowCopied] = useState(false);

  // Telegram state
  const [telegramPhoneNumber, setTelegramPhoneNumber] = useState('');
  const [telegramCode, setTelegramCode] = useState('');
  const [telegramCodeSent, setTelegramCodeSent] = useState(false);
  const [googleScopeLoading, setGoogleScopeLoading] = useState(false);
  const previousWhatsAppConnected = React.useRef<boolean | null>(null);

  // Hooks
  const { data: waStatus } = useWhatsAppStatus();
  const { data: gcalStatus } = useGCalStatus();
  const { data: gmailStatus } = useGmailStatus();
  const { data: telegramStatus } = useTelegramStatus();
  const generatePairingCode = useGeneratePairingCode();
  const sendTelegramCode = useSendTelegramCode();
  const verifyTelegramCode = useVerifyTelegramCode();
  const requestAdditionalScopes = useRequestAdditionalScopes();
  const exchangeAddScopesCode = useExchangeAddScopesCode();

  // Determine statuses
  const gmailConnectionStatus: IntegrationStatusType = !gmailStatus?.connected
    ? 'pending'
    : gmailStatus.has_scopes
      ? 'available'
      : 'needs_access';
  const gcalConnectionStatus: IntegrationStatusType = !gcalStatus?.connected
    ? 'pending'
    : gcalStatus.has_scopes
      ? 'available'
      : 'needs_access';
  const whatsappStatus: IntegrationStatusType = waStatus?.connected ? 'available' : (pairingCode ? 'connecting' : 'pending');
  const telegramStatusType: IntegrationStatusType = telegramStatus?.connected ? 'available' : (telegramCodeSent ? 'connecting' : 'pending');
  const googleScopesSelected = React.useMemo(() => {
    const scopes: ScopeType[] = [];
    if (gmailEnabled) {
      scopes.push('gmail');
    }
    if (gcalEnabled) {
      scopes.push('calendar');
    }
    return scopes;
  }, [gmailEnabled, gcalEnabled]);
  const hasGoogleIntegrationSelected = googleScopesSelected.length > 0;
  const googleScopesMissing = React.useMemo(() => {
    const scopes: ScopeType[] = [];
    if (gmailEnabled && !gmailStatus?.has_scopes) {
      scopes.push('gmail');
    }
    if (gcalEnabled && !gcalStatus?.has_scopes) {
      scopes.push('calendar');
    }
    return scopes;
  }, [gmailEnabled, gmailStatus?.has_scopes, gcalEnabled, gcalStatus?.has_scopes]);
  const googleConnected = Boolean(gmailStatus?.connected || gcalStatus?.connected);
  const googleStatus: IntegrationStatusType = googleScopeLoading
    ? 'connecting'
    : googleScopesMissing.length === 0
      ? 'available'
      : googleConnected
        ? 'needs_access'
        : 'pending';
  const googleSelectedScopeLabels = googleScopesSelected.map(getGoogleScopeLabel);
  const googleMissingScopeLabels = googleScopesMissing.map(getGoogleScopeLabel);
  const googleButtonTitle = googleScopesMissing.length > 1
    ? 'Authorize Gmail & Calendar'
    : `Authorize ${googleMissingScopeLabels[0] || 'Google'}`;

  // Check if all required integrations are available
  const allAvailable = React.useMemo(() => {
    const checks: boolean[] = [];

    if (hasGoogleIntegrationSelected) {
      checks.push(googleScopesMissing.length === 0);
    }
    if (whatsappEnabled) {
      checks.push(whatsappStatus === 'available');
    }
    if (telegramEnabled) {
      checks.push(telegramStatusType === 'available');
    }
    return checks.length > 0 && checks.every(Boolean);
  }, [
    hasGoogleIntegrationSelected,
    googleScopesMissing.length,
    whatsappEnabled,
    telegramEnabled,
    whatsappStatus,
    telegramStatusType,
  ]);

  const requiredAppsCount = React.useMemo(
    () => [gmailEnabled, gcalEnabled, whatsappEnabled, telegramEnabled].filter(Boolean).length,
    [gmailEnabled, gcalEnabled, whatsappEnabled, telegramEnabled]
  );

  const connectedAppsCount = React.useMemo(
    () =>
      [
        gmailEnabled ? gmailConnectionStatus === 'available' : false,
        gcalEnabled ? gcalConnectionStatus === 'available' : false,
        whatsappEnabled ? whatsappStatus === 'available' : false,
        telegramEnabled ? telegramStatusType === 'available' : false,
      ].filter(Boolean).length,
    [
      gmailEnabled,
      gcalEnabled,
      whatsappEnabled,
      telegramEnabled,
      gmailConnectionStatus,
      gcalConnectionStatus,
      whatsappStatus,
      telegramStatusType,
    ]
  );

  const remainingApps = React.useMemo(() => {
    const pending: string[] = [];
    if (gmailEnabled && gmailConnectionStatus !== 'available') pending.push('Gmail');
    if (gcalEnabled && gcalConnectionStatus !== 'available') pending.push('Google Calendar');
    if (whatsappEnabled && whatsappStatus !== 'available') pending.push('WhatsApp');
    if (telegramEnabled && telegramStatusType !== 'available') pending.push('Telegram');
    return pending;
  }, [
    gmailEnabled,
    gmailConnectionStatus,
    gcalEnabled,
    gcalConnectionStatus,
    whatsappEnabled,
    whatsappStatus,
    telegramEnabled,
    telegramStatusType,
  ]);

  const continueTitle = allAvailable
    ? 'Continue to configuration'
    : `Connect ${remainingApps.length} more integration${remainingApps.length === 1 ? '' : 's'}`;

  const notifyWhatsAppConnected = useCallback(async () => {
    try {
      const { status } = await Notifications.getPermissionsAsync();
      if (status !== 'granted') {
        return;
      }

      await Notifications.scheduleNotificationAsync({
        content: {
          title: 'WhatsApp Connected',
          body: 'Connection successful. Go back to Alfred to continue setup.',
          data: { screen: 'Connection' },
        },
        trigger: null,
      });
    } catch (error) {
      console.error('Failed to send WhatsApp success notification:', error);
    }
  }, []);

  // Reset pairing code when WhatsApp connects and notify only on connection transition
  useEffect(() => {
    if (!whatsappEnabled) {
      return;
    }

    const isConnected = Boolean(waStatus?.connected);
    const wasConnected = previousWhatsAppConnected.current;

    if (isConnected) {
      setPairingCode(null);
      setPhoneNumber('');
    }

    if (wasConnected === false && isConnected) {
      void notifyWhatsAppConnected();
    }

    previousWhatsAppConnected.current = isConnected;
  }, [waStatus?.connected, whatsappEnabled, notifyWhatsAppConnected]);

  // Reset Telegram state when connected
  useEffect(() => {
    if (telegramStatus?.connected) {
      setTelegramCodeSent(false);
      setTelegramPhoneNumber('');
      setTelegramCode('');
    }
  }, [telegramStatus?.connected]);

  // Reset pairing states when navigating away from this screen
  useFocusEffect(
    useCallback(() => {
      // Called when screen gains focus - nothing to do here
      return () => {
        // Called when screen loses focus - reset pairing states
        setPairingCode(null);
        setPhoneNumber('');
        setShowCopied(false);
        setTelegramCodeSent(false);
        setTelegramPhoneNumber('');
        setTelegramCode('');
      };
    }, [])
  );

  const handleConnectGoogle = async () => {
    const scopesToRequest = [...googleScopesMissing];
    if (scopesToRequest.length === 0) {
      return;
    }

    setGoogleScopeLoading(true);
    try {
      const response = await requestAdditionalScopes.mutateAsync({
        scopes: scopesToRequest,
        redirectUri: undefined,
      });

      const result = await WebBrowser.openAuthSessionAsync(response.auth_url);
      if (result.type === 'success' && result.url) {
        const codeMatch = result.url.match(/[?&]code=([^&]+)/);
        if (!codeMatch?.[1]) {
          throw new Error('No authorization code received');
        }
        await exchangeAddScopesCode.mutateAsync({
          code: decodeURIComponent(codeMatch[1]),
          scopes: scopesToRequest,
          redirectUri: undefined,
        });
      }
    } catch (error: any) {
      console.error('OAuth authorization error for Google scopes:', error);
      Alert.alert('Error', error.response?.data?.error || 'Failed to connect Google permissions');
    } finally {
      setGoogleScopeLoading(false);
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

  const handleCopyCode = async () => {
    if (pairingCode) {
      await Clipboard.setStringAsync(pairingCode);
      setShowCopied(true);
      setTimeout(() => setShowCopied(false), 2000);
    }
  };

  // Telegram handlers
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
      queryClient.invalidateQueries({ queryKey: ['telegramStatus'] });
    } catch (error: any) {
      Alert.alert('Error', error.response?.data?.error || 'Failed to verify code');
    }
  };

  const handleContinue = () => {
    // Navigate to SourceConfiguration instead of calling completeOnboarding
    navigation.navigate('SourceConfiguration', {
      whatsappEnabled,
      telegramEnabled,
      gmailEnabled,
      gcalEnabled,
    });
  };

  return (
    <SafeAreaView style={styles.safeArea} edges={['top']}>
      <KeyboardAvoidingView
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
        style={styles.keyboardAvoidingView}
      >
        <ScrollView
          style={styles.container}
          contentContainerStyle={styles.content}
          keyboardShouldPersistTaps="handled"
        >
          <Card style={styles.heroCard}>
            <View style={styles.heroTopRow}>
              <Text style={styles.step}>Step 2 of 3</Text>
              <View
                style={[
                  styles.heroStatusBadge,
                  allAvailable ? styles.heroStatusBadgeSuccess : styles.heroStatusBadgeWarning,
                ]}
              >
                <Text
                  style={[
                    styles.heroStatusText,
                    allAvailable ? styles.heroStatusTextSuccess : styles.heroStatusTextWarning,
                  ]}
                >
                  {connectedAppsCount}/{requiredAppsCount} connected
                </Text>
              </View>
            </View>
            <Text style={styles.title}>Connect Your Apps</Text>
            <Text style={styles.description}>
              Connect each selected app. Google scopes are requested together based on your selections.
            </Text>
            <View style={styles.progressTrack}>
              <View
                style={[
                  styles.progressFill,
                  { width: `${(connectedAppsCount / Math.max(requiredAppsCount, 1)) * 100}%` },
                ]}
              />
            </View>
          </Card>

        {hasGoogleIntegrationSelected && (
          <Card style={styles.card}>
            <View style={styles.integrationRow}>
              <View style={styles.integrationHeader}>
                <View style={styles.integrationInfo}>
                  <Text style={styles.integrationName}>Google</Text>
                </View>
                <View style={styles.integrationStatus}>
                  <View style={[styles.statusDot, { backgroundColor: getStatusColor(googleStatus) }]} />
                  <Text style={styles.statusLabel}>{getStatusLabel(googleStatus)}</Text>
                </View>
              </View>
              {googleScopesMissing.length > 0 && (
                <GoogleSignInButton
                  onPress={handleConnectGoogle}
                  loading={googleScopeLoading}
                  title={googleButtonTitle}
                  disabled={googleScopeLoading}
                />
              )}
            </View>
          </Card>
        )}

        {whatsappEnabled && (
          <Card style={styles.card}>
            <View style={styles.integrationRow}>
              <View style={styles.integrationHeader}>
                <View style={styles.integrationInfo}>
                  <Text style={styles.integrationName}>WhatsApp</Text>
                </View>
                <View style={styles.integrationStatus}>
                  <View style={[styles.statusDot, { backgroundColor: getStatusColor(whatsappStatus) }]} />
                  <Text style={styles.statusLabel}>{getStatusLabel(whatsappStatus)}</Text>
                </View>
              </View>

              {whatsappStatus !== 'available' && (
                <View style={styles.whatsappSection}>
                  {!pairingCode ? (
                    <>
                      <Text style={styles.phoneInputLabel}>
                        Phone number (include country code)
                      </Text>
                      <TextInput
                        style={styles.phoneInput}
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
                            <Feather
                              name={showCopied ? 'check' : 'copy'}
                              size={20}
                              color={showCopied ? colors.success : colors.primary}
                            />
                          </TouchableOpacity>
                        </View>
                      </View>
                      <Text style={styles.pairingInstructions}>
                        In WhatsApp: Settings {'>'} Linked Devices {'>'} Link with phone number
                      </Text>
                      <Text style={styles.pairingSubInstructions}>
                        Enter this code in WhatsApp to finish linking
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
          </Card>
        )}

        {telegramEnabled && (
          <Card style={styles.card}>
            <View style={styles.integrationRow}>
              <View style={styles.integrationHeader}>
                <View style={styles.integrationInfo}>
                  <Text style={styles.integrationName}>Telegram</Text>
                </View>
                <View style={styles.integrationStatus}>
                  <View style={[styles.statusDot, { backgroundColor: getStatusColor(telegramStatusType) }]} />
                  <Text style={styles.statusLabel}>{getStatusLabel(telegramStatusType)}</Text>
                </View>
              </View>

              {telegramStatusType !== 'available' && (
                <View style={styles.telegramSection}>
                  {!telegramCodeSent ? (
                    <>
                      <Text style={styles.phoneInputLabel}>
                        Phone number (include country code)
                      </Text>
                      <TextInput
                        style={styles.phoneInput}
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
                      <Text style={styles.phoneInputLabel}>
                        Enter the code sent to your Telegram app
                      </Text>
                      <TextInput
                        style={styles.phoneInput}
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
          </Card>
        )}

        <Button
          title={continueTitle}
          onPress={handleContinue}
          disabled={!allAvailable}
          style={styles.continueButton}
        />
      </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: colors.background,
  },
  keyboardAvoidingView: {
    flex: 1,
  },
  container: {
    flex: 1,
  },
  content: {
    padding: 24,
    paddingBottom: 48,
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
    fontSize: 28,
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
  heroHint: {
    marginTop: 8,
    fontSize: 12,
    color: colors.textSecondary,
  },
  card: {
    marginBottom: 16,
  },
  integrationRow: {
    paddingVertical: 4,
  },
  integrationHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  integrationInfo: {
    flex: 1,
    marginRight: 12,
  },
  integrationName: {
    fontSize: 16,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 2,
  },
  integrationStatus: {
    flexDirection: 'row',
    alignItems: 'center',
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
  googleButton: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: '#ffffff',
    borderWidth: 1,
    borderColor: '#dadce0',
    borderRadius: 8,
    paddingVertical: 12,
    paddingHorizontal: 16,
    marginTop: 16,
  },
  googleLogo: {
    width: 20,
    height: 20,
    marginRight: 12,
  },
  googleButtonText: {
    fontSize: 15,
    fontWeight: '500',
    color: '#3c4043',
  },
  whatsappSection: {
    marginTop: 16,
    paddingTop: 16,
    borderTopWidth: 1,
    borderTopColor: colors.border,
  },
  telegramSection: {
    marginTop: 16,
    paddingTop: 16,
    borderTopWidth: 1,
    borderTopColor: colors.border,
  },
  phoneInputLabel: {
    fontSize: 13,
    color: colors.textSecondary,
    marginBottom: 8,
  },
  phoneInput: {
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 8,
    paddingHorizontal: 12,
    paddingVertical: 12,
    fontSize: 16,
    color: colors.text,
    backgroundColor: colors.background,
  },
  generateButton: {
    marginTop: 12,
  },
  pairingCodeContainer: {
    alignItems: 'center',
    marginBottom: 16,
    backgroundColor: colors.card,
    borderRadius: 12,
    padding: 20,
  },
  pairingCodeLabel: {
    fontSize: 13,
    color: colors.textSecondary,
    marginBottom: 8,
  },
  pairingCodeRow: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  pairingCode: {
    fontSize: 32,
    fontWeight: '700',
    color: colors.primary,
    letterSpacing: 4,
    fontFamily: 'monospace',
  },
  copyButton: {
    padding: 8,
    marginLeft: 12,
  },
  pairingInstructions: {
    fontSize: 14,
    color: colors.text,
    textAlign: 'center',
    fontWeight: '500',
    marginBottom: 4,
  },
  pairingSubInstructions: {
    fontSize: 12,
    color: colors.textSecondary,
    textAlign: 'center',
    marginBottom: 8,
  },
  continueButton: {
    marginTop: 8,
  },
});
