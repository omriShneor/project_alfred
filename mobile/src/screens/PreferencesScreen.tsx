import React, { useState, useEffect } from 'react';
import { Text, StyleSheet, ScrollView, TouchableOpacity, View, TextInput, Alert } from 'react-native';
import * as WebBrowser from 'expo-web-browser';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useNavigation, CommonActions } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { Ionicons } from '@expo/vector-icons';
import { Card, Button } from '../components/common';
import { colors } from '../theme/colors';
import {
  useWhatsAppStatus,
  useGCalStatus,
  useDisconnectWhatsApp,
  useGetOAuthURL,
  useGeneratePairingCode,
  useTelegramStatus,
  useSendTelegramCode,
  useVerifyTelegramCode,
  useDisconnectTelegram,
} from '../hooks';
import { disconnectGCal } from '../api';
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
  const { data: waStatus, refetch: refetchWaStatus } = useWhatsAppStatus();
  const { data: gcalStatus, refetch: refetchGcalStatus } = useGCalStatus();
  const disconnectWhatsApp = useDisconnectWhatsApp();
  const getOAuthURL = useGetOAuthURL();
  const generatePairingCode = useGeneratePairingCode();

  const [disconnectingGoogle, setDisconnectingGoogle] = useState(false);
  const [showWhatsAppConnect, setShowWhatsAppConnect] = useState(false);
  const [phoneNumber, setPhoneNumber] = useState('');
  const [pairingCode, setPairingCode] = useState<string | null>(null);

  // Telegram state
  const { data: telegramStatus, refetch: refetchTelegramStatus } = useTelegramStatus();
  const sendTelegramCode = useSendTelegramCode();
  const verifyTelegramCode = useVerifyTelegramCode();
  const disconnectTelegram = useDisconnectTelegram();
  const [showTelegramConnect, setShowTelegramConnect] = useState(false);
  const [telegramPhoneNumber, setTelegramPhoneNumber] = useState('');
  const [telegramCode, setTelegramCode] = useState('');
  const [telegramCodeSent, setTelegramCodeSent] = useState(false);

  const whatsappConnected = waStatus?.connected ?? false;
  const telegramConnected = telegramStatus?.connected ?? false;
  const gmailConnected = gcalStatus?.connected ?? false; // Gmail uses same Google OAuth

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

  const handleDisconnectWhatsApp = () => {
    Alert.alert(
      'Disconnect WhatsApp',
      'Are you sure you want to disconnect WhatsApp? You will need to reconnect to scan messages.',
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

  const handleDisconnectGoogle = () => {
    Alert.alert(
      'Disconnect Google',
      'Are you sure you want to disconnect your Google account? This will disable Gmail scanning and Google Calendar sync.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Disconnect',
          style: 'destructive',
          onPress: async () => {
            setDisconnectingGoogle(true);
            try {
              await disconnectGCal();
              refetchGcalStatus();
            } catch (error) {
              Alert.alert('Error', 'Failed to disconnect Google');
            }
            setDisconnectingGoogle(false);
          },
        },
      ]
    );
  };

  const handleConnectGoogle = async () => {
    try {
      const response = await getOAuthURL.mutateAsync(undefined);
      await WebBrowser.openAuthSessionAsync(response.auth_url, 'alfred://oauth/callback');
      refetchGcalStatus();
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

  const handleShowWhatsAppConnect = () => {
    setShowWhatsAppConnect(true);
    setPairingCode(null);
    setPhoneNumber('');
  };

  // Telegram handlers
  const handleDisconnectTelegram = () => {
    Alert.alert(
      'Disconnect Telegram',
      'Are you sure you want to disconnect Telegram? You will need to reconnect to scan messages.',
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
        {/* Sources Section */}
        <Text style={styles.sectionLabel}>Sources</Text>
        <Text style={styles.sectionDescription}>
            Select where Alfred should look for event suggestions.
        </Text>

        {whatsappConnected && (
          <PreferenceCard
            title="WhatsApp"
            description="Manage tracked contacts and groups"
            icon="chatbubble-outline"
            connected={whatsappConnected}
            onPress={() => navigation.navigate('WhatsAppPreferences')}
          />
        )}

        {telegramConnected && (
          <PreferenceCard
            title="Telegram"
            description="Manage tracked contacts and groups"
            icon="paper-plane-outline"
            connected={telegramConnected}
            onPress={() => navigation.navigate('TelegramPreferences')}
          />
        )}

        {gmailConnected && (
          <PreferenceCard
            title="Gmail"
            description="Manage tracked senders and domains"
            icon="mail-outline"
            connected={gmailConnected}
            onPress={() => navigation.navigate('GmailPreferences')}
          />
        )}

        {!whatsappConnected && !telegramConnected && !gmailConnected && (
          <Card style={styles.emptyCard}>
            <Text style={styles.emptyText}>
              No connected sources
            </Text>
            <Text style={styles.emptySubtext}>
              Connect an account below to start scanning for events
            </Text>
          </Card>
        )}

        {/* Connected Accounts Section */}
        <Text style={styles.sectionLabel}>Connected Accounts</Text>
        <Card>
          {/* WhatsApp */}
          <View>
            <View style={styles.accountRow}>
              <View style={styles.accountInfo}>
                <Ionicons name="chatbubble-outline" size={20} color={colors.text} />
                <View style={styles.accountText}>
                  <Text style={styles.accountName}>WhatsApp</Text>
                  <Text style={styles.accountStatus}>
                    {whatsappConnected ? 'Connected' : 'Not connected'}
                  </Text>
                </View>
              </View>
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
                  title="Connect"
                  onPress={handleShowWhatsAppConnect}
                  style={styles.connectButton}
                />
              ) : null}
            </View>
            {!whatsappConnected && showWhatsAppConnect && (
              <View style={styles.whatsappConnectSection}>
                {!pairingCode ? (
                  <>
                    <Text style={styles.connectLabel}>
                      Enter your phone number with country code
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
                      <Text style={styles.pairingCodeLabel}>Your Pairing Code</Text>
                      <Text style={styles.pairingCode}>{pairingCode}</Text>
                    </View>
                    <Text style={styles.pairingInstructions}>
                      Open WhatsApp → Settings → Linked Devices → Link with phone number
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

          {/* Telegram */}
          <View style={[styles.accountRow, styles.accountRowBorder]}>
            <View style={styles.accountInfo}>
              <Ionicons name="paper-plane-outline" size={20} color={colors.text} />
              <View style={styles.accountText}>
                <Text style={styles.accountName}>Telegram</Text>
                <Text style={styles.accountStatus}>
                  {telegramConnected ? 'Connected' : 'Not connected'}
                </Text>
              </View>
            </View>
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
                title="Connect"
                onPress={handleShowTelegramConnect}
                style={styles.connectButton}
              />
            ) : null}
          </View>
          {!telegramConnected && showTelegramConnect && (
            <View style={styles.telegramConnectSection}>
              {!telegramCodeSent ? (
                <>
                  <Text style={styles.connectLabel}>
                    Enter your phone number with country code
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
                    Enter the verification code sent to Telegram
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

          {/* Google Account */}
          <View style={[styles.accountRow, styles.accountRowBorder]}>
            <View style={styles.accountInfo}>
              <Ionicons name="logo-google" size={20} color={colors.text} />
              <View style={styles.accountText}>
                <Text style={styles.accountName}>Google Account</Text>
                <Text style={styles.accountStatus}>
                  {gmailConnected ? 'Connected' : 'Not connected'}
                </Text>
              </View>
            </View>
            {gmailConnected ? (
              <Button
                title="Disconnect"
                variant="outline"
                onPress={handleDisconnectGoogle}
                loading={disconnectingGoogle}
                style={styles.disconnectButton}
              />
            ) : (
              <Button
                title="Connect"
                onPress={handleConnectGoogle}
                loading={getOAuthURL.isPending}
                style={styles.connectButton}
              />
            )}
          </View>
        </Card>

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
    paddingBottom: 32,
  },
  sectionLabel: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.textSecondary,
    marginTop: 16,
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
    paddingTop: 12,
    borderTopWidth: 1,
    borderTopColor: colors.border,
  },
  telegramConnectSection: {
    marginTop: 12,
    paddingTop: 12,
    paddingHorizontal: 16,
    borderTopWidth: 1,
    borderTopColor: colors.border,
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
  pairingCode: {
    fontSize: 28,
    fontWeight: '700',
    color: colors.primary,
    letterSpacing: 4,
  },
  pairingInstructions: {
    fontSize: 13,
    color: colors.textSecondary,
    textAlign: 'center',
    marginBottom: 12,
  },
  noAccountsText: {
    fontSize: 14,
    color: colors.textSecondary,
    textAlign: 'center',
    paddingVertical: 16,
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
  settingsButton: {
    minWidth: 150,
  },
});
