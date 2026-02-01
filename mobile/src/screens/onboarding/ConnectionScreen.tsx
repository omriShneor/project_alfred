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
import { useRoute, useFocusEffect } from '@react-navigation/native';
import type { RouteProp } from '@react-navigation/native';
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
  useGetOAuthURL,
  useCompleteOnboarding,
  useTelegramStatus,
  useSendTelegramCode,
  useVerifyTelegramCode,
} from '../../hooks';
import type { OnboardingParamList } from '../../navigation/OnboardingNavigator';

type RouteProps = RouteProp<OnboardingParamList, 'Connection'>;
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
        source={{ uri: 'https://developers.google.com/identity/images/g-logo.png' }}
        style={styles.googleLogo}
      />
      <Text style={styles.googleButtonText}>
        {loading ? 'Connecting...' : 'Sign in with Google'}
      </Text>
    </TouchableOpacity>
  );
}

export function ConnectionScreen() {
  const route = useRoute<RouteProps>();
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
  const getOAuthURL = useGetOAuthURL();
  const completeOnboarding = useCompleteOnboarding();
  const sendTelegramCode = useSendTelegramCode();
  const verifyTelegramCode = useVerifyTelegramCode();

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
      const response = await getOAuthURL.mutateAsync(undefined);
      await WebBrowser.openAuthSessionAsync(response.auth_url, 'alfred://oauth/success');
      queryClient.invalidateQueries({ queryKey: ['gcalStatus'] });
    } catch (error: any) {
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
        <Text style={styles.step}>Step 2 of 2</Text>
        <Text style={styles.title}>Connect Your Accounts</Text>
        <Text style={styles.description}>
          Connect the services you selected to start scanning for events.
        </Text>

        {gmailEnabled && (
          <Card style={styles.card}>
            <View style={styles.integrationRow}>
              <View style={styles.integrationHeader}>
                <View style={styles.integrationInfo}>
                  <Text style={styles.integrationName}>Google Account</Text>
                  <Text style={styles.integrationDescription}>For Gmail access</Text>
                </View>
                <View style={styles.integrationStatus}>
                  <View style={[styles.statusDot, { backgroundColor: getStatusColor(googleStatus) }]} />
                  <Text style={styles.statusLabel}>{getStatusLabel(googleStatus)}</Text>
                </View>
              </View>
              {googleStatus !== 'available' && (
                <GoogleSignInButton
                  onPress={handleConnectGoogle}
                  loading={getOAuthURL.isPending}
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
                  <Text style={styles.integrationDescription}>For message scanning</Text>
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
                        Enter your phone number with country code
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
                        <Text style={styles.pairingCodeLabel}>Your Pairing Code</Text>
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
                        Open WhatsApp {'>'} Settings {'>'} Linked Devices {'>'} Link with phone number
                      </Text>
                      <Text style={styles.pairingSubInstructions}>
                        Enter this 8-digit code in WhatsApp
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
                  <Text style={styles.integrationDescription}>For message scanning</Text>
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
                        Enter your phone number with country code
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
                        Enter the verification code sent to Telegram
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

        {!allAvailable && (
          <View style={styles.statusSummary}>
            <Feather name="info" size={16} color={colors.textSecondary} />
            <Text style={styles.statusSummaryText}>
              Connect all services above to continue
            </Text>
          </View>
        )}

        <Button
          title="Continue"
          onPress={handleContinue}
          disabled={!allAvailable}
          loading={completeOnboarding.isPending}
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
  integrationDescription: {
    fontSize: 13,
    color: colors.textSecondary,
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
  statusSummary: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: 8,
    marginBottom: 16,
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
