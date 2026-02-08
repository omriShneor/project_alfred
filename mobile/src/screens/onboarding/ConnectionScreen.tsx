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
import { Feather } from '@expo/vector-icons';
import { Button, Card } from '../../components/common';
import { colors } from '../../theme/colors';
import { useQueryClient } from '@tanstack/react-query';
import {
  useWhatsAppStatus,
  useGCalStatus,
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
type IntegrationStatusType = 'pending' | 'connecting' | 'available' | 'error';

function getStatusColor(status: IntegrationStatusType) {
  switch (status) {
    case 'available':
      return colors.success;
    case 'connecting':
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
    case 'error':
      return 'Error';
    default:
      return 'Not connected';
  }
}

function GoogleSignInButton({ onPress, loading }: { onPress: () => void; loading?: boolean }) {
  return (
    <TouchableOpacity
      style={styles.googleButton}
      onPress={onPress}
      disabled={loading}
      activeOpacity={0.7}
    >
      <Image
        source={require('../../../assets/google-logo.png')}
        style={styles.googleLogo}
      />
      <Text style={styles.googleButtonText}>
        {loading ? 'Connecting...' : 'Connect Google'}
      </Text>
    </TouchableOpacity>
  );
}

export function ConnectionScreen() {
  const route = useRoute<RouteProps>();
  const navigation = useNavigation<NavigationProp>();
  const queryClient = useQueryClient();

  const { whatsappEnabled, telegramEnabled, gmailEnabled } = route.params;

  // WhatsApp state
  const [phoneNumber, setPhoneNumber] = useState('');
  const [pairingCode, setPairingCode] = useState<string | null>(null);
  const [showCopied, setShowCopied] = useState(false);

  // Telegram state
  const [telegramPhoneNumber, setTelegramPhoneNumber] = useState('');
  const [telegramCode, setTelegramCode] = useState('');
  const [telegramCodeSent, setTelegramCodeSent] = useState(false);

  // Hooks
  const { data: waStatus } = useWhatsAppStatus();
  const { data: gcalStatus } = useGCalStatus();
  const { data: telegramStatus } = useTelegramStatus();
  const generatePairingCode = useGeneratePairingCode();
  const sendTelegramCode = useSendTelegramCode();
  const verifyTelegramCode = useVerifyTelegramCode();
  const requestAdditionalScopes = useRequestAdditionalScopes();
  const exchangeAddScopesCode = useExchangeAddScopesCode();

  // Determine statuses
  const googleStatus: IntegrationStatusType = gcalStatus?.connected ? 'available' : 'pending';
  const whatsappStatus: IntegrationStatusType = waStatus?.connected ? 'available' : (pairingCode ? 'connecting' : 'pending');
  const telegramStatusType: IntegrationStatusType = telegramStatus?.connected ? 'available' : (telegramCodeSent ? 'connecting' : 'pending');

  // Check if all required integrations are available
  const allAvailable = React.useMemo(() => {
    const checks: boolean[] = [];

    if (gmailEnabled) {
      checks.push(googleStatus === 'available');
    }
    if (whatsappEnabled) {
      checks.push(whatsappStatus === 'available');
    }
    if (telegramEnabled) {
      checks.push(telegramStatusType === 'available');
    }

    return checks.length > 0 && checks.every(Boolean);
  }, [gmailEnabled, whatsappEnabled, telegramEnabled, googleStatus, whatsappStatus, telegramStatusType]);

  const requiredAppsCount = React.useMemo(
    () => [gmailEnabled, whatsappEnabled, telegramEnabled].filter(Boolean).length,
    [gmailEnabled, whatsappEnabled, telegramEnabled]
  );

  const connectedAppsCount = React.useMemo(
    () =>
      [
        gmailEnabled ? googleStatus === 'available' : false,
        whatsappEnabled ? whatsappStatus === 'available' : false,
        telegramEnabled ? telegramStatusType === 'available' : false,
      ].filter(Boolean).length,
    [
      gmailEnabled,
      whatsappEnabled,
      telegramEnabled,
      googleStatus,
      whatsappStatus,
      telegramStatusType,
    ]
  );

  const remainingApps = React.useMemo(() => {
    const pending: string[] = [];
    if (gmailEnabled && googleStatus !== 'available') pending.push('Google');
    if (whatsappEnabled && whatsappStatus !== 'available') pending.push('WhatsApp');
    if (telegramEnabled && telegramStatusType !== 'available') pending.push('Telegram');
    return pending;
  }, [
    gmailEnabled,
    googleStatus,
    whatsappEnabled,
    whatsappStatus,
    telegramEnabled,
    telegramStatusType,
  ]);

  const continueTitle = allAvailable
    ? 'Continue to choose contacts'
    : `Connect ${remainingApps.length} more app${remainingApps.length === 1 ? '' : 's'}`;

  const heroHint = allAvailable
    ? 'All selected apps are connected. Continue to choose contacts and senders.'
    : `Remaining: ${remainingApps.join(', ')}`;

  // Reset pairing code when WhatsApp connects
  useEffect(() => {
    if (waStatus?.connected) {
      setPairingCode(null);
      setPhoneNumber('');
    }
  }, [waStatus?.connected]);

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
    try {
      // Request both Gmail and Calendar scopes together
      const response = await requestAdditionalScopes.mutateAsync({
        scopes: ['gmail' as ScopeType, 'calendar' as ScopeType],
        redirectUri: undefined // Use default HTTPS callback (will be /api/auth/callback)
      });

      // Don't specify redirect URL - let the OAuth flow complete naturally through the backend
      const result = await WebBrowser.openAuthSessionAsync(response.auth_url);

      // Handle the OAuth callback directly since WebBrowser captures the deep link
      if (result.type === 'success' && result.url) {
        const url = result.url;

        // Extract code from URL
        const codeMatch = url.match(/[?&]code=([^&]+)/);
        if (codeMatch && codeMatch[1]) {
          const code = decodeURIComponent(codeMatch[1]);

          try {
            await exchangeAddScopesCode.mutateAsync({
              code,
              scopes: ['gmail' as ScopeType, 'calendar' as ScopeType],
              redirectUri: undefined
            });
          } catch (error: any) {
            console.error('Failed to exchange OAuth code:', error);
            Alert.alert('Error', 'Failed to connect Google account. Please try again.');
          }
        }
      }
    } catch (error: any) {
      console.error('OAuth authorization error:', error);
      Alert.alert('Error', error.response?.data?.error || 'Failed to start Google authorization');
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
              Connect each selected app so Alfred can start detecting events, reminders, and tasks.
            </Text>
            <View style={styles.progressTrack}>
              <View
                style={[
                  styles.progressFill,
                  { width: `${(connectedAppsCount / Math.max(requiredAppsCount, 1)) * 100}%` },
                ]}
              />
            </View>
            <Text style={styles.heroHint}>{heroHint}</Text>
          </Card>

        {gmailEnabled && (
          <Card style={styles.card}>
            <View style={styles.integrationRow}>
              <View style={styles.integrationHeader}>
                <View style={styles.integrationInfo}>
                  <Text style={styles.integrationName}>Google (Gmail & Calendar)</Text>
                </View>
                <View style={styles.integrationStatus}>
                  <View style={[styles.statusDot, { backgroundColor: getStatusColor(googleStatus) }]} />
                  <Text style={styles.statusLabel}>{getStatusLabel(googleStatus)}</Text>
                </View>
              </View>
              {googleStatus !== 'available' && (
                <GoogleSignInButton
                  onPress={handleConnectGoogle}
                  loading={requestAdditionalScopes.isPending || exchangeAddScopesCode.isPending}
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
