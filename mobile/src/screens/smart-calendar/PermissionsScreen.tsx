import React, { useState, useEffect } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  Alert,
  TouchableOpacity,
  Image,
  TextInput,
} from 'react-native';
import { useNavigation } from '@react-navigation/native';
import * as WebBrowser from 'expo-web-browser';
import * as Clipboard from 'expo-clipboard';
import { Feather } from '@expo/vector-icons';
import { Button, Card } from '../../components/common';
import { colors } from '../../theme/colors';
import { useQueryClient } from '@tanstack/react-query';
import {
  useSmartCalendarStatus,
  useFeatures,
  useUpdateSmartCalendar,
  useWhatsAppStatus,
  useGCalStatus,
  useGeneratePairingCode,
  useGetOAuthURL,
} from '../../hooks';
import type { DrawerNavigationProp } from '@react-navigation/drawer';
import type { DrawerParamList } from '../../navigation/DrawerNavigator';

type NavigationProp = DrawerNavigationProp<DrawerParamList>;

type IntegrationStatusType = 'pending' | 'connecting' | 'available' | 'error';

interface IntegrationRowProps {
  name: string;
  description: string;
  status: IntegrationStatusType;
  onConnect: () => void;
  isConnecting?: boolean;
  children?: React.ReactNode;
  customButton?: React.ReactNode;
}

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
      return 'Available';
    case 'connecting':
      return 'Connecting...';
    case 'error':
      return 'Error';
    default:
      return 'Pending';
  }
}

function IntegrationRow({ name, description, status, onConnect, isConnecting, children, customButton }: IntegrationRowProps) {
  const showConnectButton = status === 'pending' || status === 'error';

  return (
    <View style={styles.integrationRow}>
      <View style={styles.integrationHeader}>
        <View style={styles.integrationInfo}>
          <Text style={styles.integrationName}>{name}</Text>
          <Text style={styles.integrationDescription}>{description}</Text>
        </View>
        <View style={styles.integrationStatus}>
          <View style={[styles.statusDot, { backgroundColor: getStatusColor(status) }]} />
          <Text style={styles.statusLabel}>{getStatusLabel(status)}</Text>
        </View>
      </View>
      {showConnectButton && (
        customButton || (
          <Button
            title="Connect"
            onPress={onConnect}
            variant="outline"
            loading={isConnecting}
            style={styles.connectButton}
          />
        )
      )}
      {children}
    </View>
  );
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

export function SmartCalendarPermissionsScreen() {
  const navigation = useNavigation<NavigationProp>();
  const queryClient = useQueryClient();
  const [phoneNumber, setPhoneNumber] = useState('');
  const [pairingCode, setPairingCode] = useState<string | null>(null);
  const [showCopied, setShowCopied] = useState(false);

  const { data: features } = useFeatures();
  useSmartCalendarStatus();
  const { data: waStatus } = useWhatsAppStatus();
  const { data: gcalStatus } = useGCalStatus();
  const updateSmartCalendar = useUpdateSmartCalendar();
  const generatePairingCode = useGeneratePairingCode();
  const getOAuthURL = useGetOAuthURL();

  // Get enabled inputs/calendars from features
  const inputs = features?.smart_calendar?.inputs;
  const calendars = features?.smart_calendar?.calendars;

  // Determine statuses
  const googleCalendarStatus: IntegrationStatusType = gcalStatus?.connected ? 'available' : 'pending';
  const whatsappStatus: IntegrationStatusType = waStatus?.connected ? 'available' : (pairingCode ? 'connecting' : 'pending');

  // Determine which integrations are actually needed
  const needsGoogle = calendars?.google_calendar?.enabled || inputs?.email?.enabled;
  const needsWhatsApp = inputs?.whatsapp?.enabled;

  // If only Alfred calendar is selected (no external integrations needed)
  const noExternalAuthNeeded = !needsGoogle && !needsWhatsApp;

  // Check if all required integrations are available
  const allAvailable = React.useMemo(() => {
    // If no external auth is needed, we're always ready
    if (noExternalAuthNeeded) {
      return true;
    }

    let required: boolean[] = [];

    if (needsGoogle) {
      required.push(googleCalendarStatus === 'available');
    }
    if (needsWhatsApp) {
      required.push(whatsappStatus === 'available');
    }

    return required.length > 0 && required.every(Boolean);
  }, [noExternalAuthNeeded, needsGoogle, needsWhatsApp, googleCalendarStatus, whatsappStatus]);

  // Reset pairing code when WhatsApp connects
  useEffect(() => {
    if (waStatus?.connected) {
      setPairingCode(null);
      setPhoneNumber('');
    }
  }, [waStatus?.connected]);

  const handleConnectGoogle = async () => {
    try {
      // Don't send redirect_uri - backend will use its own HTTPS callback
      const response = await getOAuthURL.mutateAsync(undefined);

      // Open auth session - it will automatically close when alfred:// deep link is triggered
      // Backend handles OAuth callback and redirects to alfred://oauth/success
      await WebBrowser.openAuthSessionAsync(response.auth_url, 'alfred://oauth/success');

      // Invalidate status queries to trigger immediate refresh after OAuth completes
      queryClient.invalidateQueries({ queryKey: ['gcalStatus'] });
      queryClient.invalidateQueries({ queryKey: ['onboardingStatus'] });
    } catch (error: any) {
      Alert.alert(
        'Error',
        error.response?.data?.error || 'Failed to start Google authorization'
      );
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
      Alert.alert(
        'Error',
        error.response?.data?.error || 'Failed to generate pairing code'
      );
    }
  };

  const handleContinue = async () => {
    try {
      // Mark setup as complete
      await updateSmartCalendar.mutateAsync({ setup_complete: true });

      // Navigate to Home screen after setup completion
      // Using getParent() to access the drawer navigator from within the stack
      const parent = navigation.getParent();
      if (parent) {
        parent.navigate('Home');
      } else {
        navigation.navigate('Home' as any);
      }
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to complete setup');
    }
  };

  // If no external auth is needed, show a simple confirmation screen
  if (noExternalAuthNeeded) {
    return (
      <ScrollView style={styles.container} contentContainerStyle={styles.content}>
        <View style={styles.noAuthContainer}>
          <Feather name="check-circle" size={48} color={colors.success} />
          <Text style={styles.noAuthTitle}>Ready to go!</Text>
          <Text style={styles.noAuthDescription}>
            You've selected Alfred Calendar which stores events locally.{'\n'}
            No additional account connections are needed.
          </Text>
        </View>

        <Button
          title="Complete Setup"
          onPress={handleContinue}
          loading={updateSmartCalendar.isPending}
          style={styles.continueButton}
        />
      </ScrollView>
    );
  }

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <Text style={styles.headerText}>
        Connect required services to enable Smart Calendar
      </Text>

      {/* Google Account - Required for both Calendar and Gmail */}
      {needsGoogle && (
        <Card style={styles.card}>
          <IntegrationRow
            name="Google Account"
            description={
              inputs?.email?.enabled && calendars?.google_calendar?.enabled
                ? "For Calendar and Gmail access"
                : inputs?.email?.enabled
                ? "For Gmail access"
                : "For Calendar access"
            }
            status={googleCalendarStatus}
            onConnect={handleConnectGoogle}
            isConnecting={getOAuthURL.isPending}
            customButton={
              <GoogleSignInButton
                onPress={handleConnectGoogle}
                loading={getOAuthURL.isPending}
              />
            }
          />
        </Card>
      )}

      {/* WhatsApp */}
      {needsWhatsApp && (
        <Card style={styles.card}>
          <IntegrationRow
            name="WhatsApp"
            description="For message scanning"
            status={whatsappStatus}
            onConnect={handleConnectWhatsApp}
            isConnecting={generatePairingCode.isPending}
            customButton={<></>}
          >
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
                        <TouchableOpacity
                          style={styles.copyButton}
                          onPress={() => {
                            if (pairingCode) {
                              Clipboard.setStringAsync(pairingCode);
                              setShowCopied(true);
                              setTimeout(() => setShowCopied(false), 2000);
                            }
                          }}
                        >
                          <Feather name={showCopied ? "check" : "copy"} size={20} color={showCopied ? colors.success : colors.primary} />
                        </TouchableOpacity>
                        {showCopied && (
                          <Text style={styles.copiedText}>Copied!</Text>
                        )}
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
          </IntegrationRow>
        </Card>
      )}

      {/* Status summary */}
      {!allAvailable && (
        <View style={styles.statusSummary}>
          <Feather name="info" size={16} color={colors.textSecondary} />
          <Text style={styles.statusSummaryText}>
            Connect all services above to continue
          </Text>
        </View>
      )}

      {/* Continue Button */}
      <Button
        title="Continue"
        onPress={handleContinue}
        disabled={!allAvailable}
        loading={updateSmartCalendar.isPending}
        style={styles.continueButton}
      />
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
    paddingBottom: 32,
  },
  headerText: {
    fontSize: 15,
    color: colors.textSecondary,
    textAlign: 'center',
    marginBottom: 24,
    lineHeight: 22,
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
  connectButton: {
    marginTop: 12,
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
    marginTop: 12,
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
    marginLeft: 12,
    padding: 8,
  },
  copiedText: {
    marginLeft: 8,
    fontSize: 13,
    color: colors.success,
    fontWeight: '500',
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
  noAuthContainer: {
    alignItems: 'center',
    paddingVertical: 48,
  },
  noAuthTitle: {
    fontSize: 20,
    fontWeight: '600',
    color: colors.text,
    marginTop: 16,
    marginBottom: 8,
  },
  noAuthDescription: {
    fontSize: 14,
    color: colors.textSecondary,
    textAlign: 'center',
    lineHeight: 20,
  },
});
