import React, { useState, useEffect, useCallback } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TextInput,
  Switch,
  Alert,
  Linking,
  Platform,
} from 'react-native';
import * as WebBrowser from 'expo-web-browser';
import * as ExpoLinking from 'expo-linking';
import { Button, Card, LoadingSpinner } from '../components/common';
import { colors } from '../theme/colors';
import {
  useWhatsAppStatus,
  useGCalStatus,
  useGeneratePairingCode,
  useDisconnectWhatsApp,
  useGetOAuthURL,
  useExchangeOAuthCode,
  usePushNotifications,
} from '../hooks';
import { getNotificationPrefs, updateEmailPrefs, updatePushPrefs } from '../api';

export function SettingsScreen() {
  const [phoneNumber, setPhoneNumber] = useState('');
  const [pairingCode, setPairingCode] = useState<string | null>(null);
  const [emailEnabled, setEmailEnabled] = useState(false);
  const [emailAddress, setEmailAddress] = useState('');
  const [emailAvailable, setEmailAvailable] = useState(false);
  const [pushEnabled, setPushEnabled] = useState(false);
  const [pushAvailable, setPushAvailable] = useState(false);
  const [savingEmail, setSavingEmail] = useState(false);
  const [savingPush, setSavingPush] = useState(false);

  const { data: waStatus, isLoading: waLoading, refetch: refetchWA } = useWhatsAppStatus();
  const { data: gcalStatus, isLoading: gcalLoading, refetch: refetchGCal } = useGCalStatus();
  const generateCode = useGeneratePairingCode();
  const disconnectWA = useDisconnectWhatsApp();
  const getOAuthURL = useGetOAuthURL();
  const exchangeCode = useExchangeOAuthCode();

  const {
    expoPushToken,
    permissionStatus,
    isRegistering,
    requestPermissions,
  } = usePushNotifications();

  useEffect(() => {
    loadNotificationPrefs();
  }, []);

  // Reset pairing code when WhatsApp connects
  useEffect(() => {
    if (waStatus?.connected) {
      setPairingCode(null);
    }
  }, [waStatus?.connected]);

  const loadNotificationPrefs = async () => {
    try {
      const prefs = await getNotificationPrefs();
      setEmailEnabled(prefs.preferences.email_enabled);
      setEmailAddress(prefs.preferences.email_address || '');
      setEmailAvailable(prefs.available.email);
      setPushEnabled(prefs.preferences.push_enabled);
      setPushAvailable(prefs.available.push);
    } catch (error) {
      // Ignore errors
    }
  };

  const handleGeneratePairingCode = async () => {
    if (!phoneNumber.trim()) {
      Alert.alert('Error', 'Please enter your phone number');
      return;
    }

    const formattedNumber = phoneNumber.startsWith('+')
      ? phoneNumber
      : `+${phoneNumber}`;

    try {
      const result = await generateCode.mutateAsync(formattedNumber);
      setPairingCode(result.code);
    } catch (error: any) {
      Alert.alert(
        'Error',
        error.response?.data?.error || 'Failed to generate pairing code'
      );
    }
  };

  const handleDisconnectWhatsApp = () => {
    Alert.alert(
      'Disconnect WhatsApp',
      'Are you sure you want to disconnect WhatsApp? You will need to pair again.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Disconnect',
          style: 'destructive',
          onPress: async () => {
            try {
              await disconnectWA.mutateAsync();
              setPairingCode(null);
              refetchWA();
            } catch (error: any) {
              Alert.alert('Error', 'Failed to disconnect WhatsApp');
            }
          },
        },
      ]
    );
  };

  const handleOAuthCallback = useCallback(
    async (code: string) => {
      const redirectUri = ExpoLinking.createURL('oauth/callback');
      try {
        await exchangeCode.mutateAsync({ code, redirectUri });
        refetchGCal();
      } catch (error: any) {
        Alert.alert(
          'Error',
          error.response?.data?.error || 'Failed to connect Google Calendar'
        );
      }
    },
    [exchangeCode, refetchGCal]
  );

  // Listen for deep link callback
  useEffect(() => {
    const handleUrl = ({ url }: { url: string }) => {
      const parsed = ExpoLinking.parse(url);
      if (parsed.path === 'oauth/callback' && parsed.queryParams?.code) {
        handleOAuthCallback(parsed.queryParams.code as string);
      }
    };

    const subscription = Linking.addEventListener('url', handleUrl);
    return () => subscription.remove();
  }, [handleOAuthCallback]);

  const handleConnectGoogle = async () => {
    const redirectUri = ExpoLinking.createURL('oauth/callback');

    try {
      const response = await getOAuthURL.mutateAsync(redirectUri);

      const result = await WebBrowser.openAuthSessionAsync(
        response.auth_url,
        redirectUri
      );

      if (result.type === 'success' && result.url) {
        const parsed = ExpoLinking.parse(result.url);
        if (parsed.queryParams?.code) {
          await handleOAuthCallback(parsed.queryParams.code as string);
        }
      }
    } catch (error: any) {
      Alert.alert(
        'Error',
        error.response?.data?.error || 'Failed to start Google authorization'
      );
    }
  };

  const handleSaveEmailPrefs = async () => {
    if (emailEnabled && !emailAddress.trim()) {
      Alert.alert('Error', 'Please enter an email address');
      return;
    }

    setSavingEmail(true);
    try {
      await updateEmailPrefs(emailEnabled, emailAddress.trim());
      Alert.alert('Success', 'Email preferences saved');
    } catch (error: any) {
      Alert.alert('Error', 'Failed to save email preferences');
    }
    setSavingEmail(false);
  };

  const handlePushToggle = async (enabled: boolean) => {
    // If enabling and no token, request permissions first
    if (enabled && !expoPushToken) {
      if (permissionStatus === 'denied') {
        Alert.alert(
          'Permission Required',
          'Push notifications are disabled. Please enable them in your device settings.',
          [{ text: 'OK' }]
        );
        return;
      }

      setSavingPush(true);
      const success = await requestPermissions();
      setSavingPush(false);

      if (success) {
        setPushEnabled(true);
      }
      return;
    }

    // Update backend preference
    setSavingPush(true);
    try {
      await updatePushPrefs(enabled);
      setPushEnabled(enabled);
    } catch (error: any) {
      Alert.alert('Error', 'Failed to update push preferences');
    }
    setSavingPush(false);
  };

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      {/* WhatsApp Section */}
      <Text style={styles.sectionTitle}>WhatsApp</Text>
      <Card>
        <View style={styles.statusRow}>
          <View style={styles.statusInfo}>
            <Text style={styles.statusLabel}>Status</Text>
            {waLoading ? (
              <LoadingSpinner size="small" />
            ) : (
              <View style={styles.statusBadge}>
                <View
                  style={[
                    styles.statusDot,
                    { backgroundColor: waStatus?.connected ? colors.success : colors.danger },
                  ]}
                />
                <Text style={styles.statusText}>
                  {waStatus?.connected ? 'Connected' : 'Not Connected'}
                </Text>
              </View>
            )}
          </View>
        </View>

        {waStatus?.connected ? (
          <Button
            title="Disconnect"
            onPress={handleDisconnectWhatsApp}
            variant="danger"
            loading={disconnectWA.isPending}
            style={styles.actionButton}
          />
        ) : pairingCode ? (
          <View>
            <View style={styles.codeDisplay}>
              <Text style={styles.codeLabel}>Pairing Code</Text>
              <Text style={styles.code}>{pairingCode}</Text>
            </View>
            <Text style={styles.codeInstructions}>
              Enter this code in WhatsApp {'>'} Linked Devices {'>'} Link with phone number
            </Text>
            <Button
              title="Generate New Code"
              onPress={handleGeneratePairingCode}
              variant="outline"
              loading={generateCode.isPending}
              style={styles.actionButton}
            />
          </View>
        ) : (
          <View>
            <TextInput
              style={styles.input}
              value={phoneNumber}
              onChangeText={setPhoneNumber}
              placeholder="Phone number (e.g., +1234567890)"
              keyboardType="phone-pad"
            />
            <Button
              title="Generate Pairing Code"
              onPress={handleGeneratePairingCode}
              loading={generateCode.isPending}
              disabled={!phoneNumber.trim()}
              style={styles.actionButton}
            />
          </View>
        )}
      </Card>

      {/* Google Calendar Section */}
      <Text style={styles.sectionTitle}>Google Calendar</Text>
      <Card>
        <View style={styles.statusRow}>
          <View style={styles.statusInfo}>
            <Text style={styles.statusLabel}>Status</Text>
            {gcalLoading ? (
              <LoadingSpinner size="small" />
            ) : (
              <View style={styles.statusBadge}>
                <View
                  style={[
                    styles.statusDot,
                    { backgroundColor: gcalStatus?.connected ? colors.success : colors.danger },
                  ]}
                />
                <Text style={styles.statusText}>
                  {gcalStatus?.connected ? 'Connected' : 'Not Connected'}
                </Text>
              </View>
            )}
          </View>
        </View>

        {!gcalStatus?.connected && (
          <Button
            title="Connect Google Calendar"
            onPress={handleConnectGoogle}
            loading={getOAuthURL.isPending || exchangeCode.isPending}
            style={styles.actionButton}
          />
        )}
      </Card>

      {/* Notifications Section */}
      <Text style={styles.sectionTitle}>Notifications</Text>
      <Card>
        <View style={styles.settingRow}>
          <View style={styles.settingInfo}>
            <Text style={styles.settingTitle}>Email Notifications</Text>
            <Text style={styles.settingDescription}>
              Get notified when events are detected
            </Text>
            {!emailAvailable && (
              <Text style={styles.unavailableText}>
                Email not configured on server
              </Text>
            )}
          </View>
          <Switch
            value={emailEnabled}
            onValueChange={setEmailEnabled}
            disabled={!emailAvailable}
            trackColor={{ false: colors.border, true: colors.primary }}
            thumbColor="#ffffff"
          />
        </View>

        {emailEnabled && (
          <View style={styles.emailSection}>
            <TextInput
              style={styles.input}
              value={emailAddress}
              onChangeText={setEmailAddress}
              placeholder="your@email.com"
              keyboardType="email-address"
              autoCapitalize="none"
            />
            <Button
              title="Save"
              onPress={handleSaveEmailPrefs}
              loading={savingEmail}
              size="small"
              style={styles.saveButton}
            />
          </View>
        )}
      </Card>

      {/* Push Notifications */}
      {Platform.OS !== 'web' && pushAvailable && (
        <Card style={styles.pushCard}>
          <View style={styles.settingRow}>
            <View style={styles.settingInfo}>
              <Text style={styles.settingTitle}>Push Notifications</Text>
              <Text style={styles.settingDescription}>
                Get instant alerts on your phone
              </Text>
              {permissionStatus === 'denied' && (
                <Text style={styles.unavailableText}>
                  Permission denied - enable in device settings
                </Text>
              )}
            </View>
            {(savingPush || isRegistering) ? (
              <LoadingSpinner size="small" />
            ) : (
              <Switch
                value={pushEnabled}
                onValueChange={handlePushToggle}
                disabled={!pushAvailable}
                trackColor={{ false: colors.border, true: colors.primary }}
                thumbColor="#ffffff"
              />
            )}
          </View>
        </Card>
      )}

      {/* About Section */}
      <Text style={styles.sectionTitle}>About</Text>
      <Card>
        <View style={styles.aboutRow}>
          <Text style={styles.aboutLabel}>Version</Text>
          <Text style={styles.aboutValue}>1.0.0</Text>
        </View>
        <View style={styles.aboutRow}>
          <Text style={styles.aboutLabel}>App</Text>
          <Text style={styles.aboutValue}>Project Alfred</Text>
        </View>
      </Card>
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
  sectionTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.textSecondary,
    marginTop: 16,
    marginBottom: 8,
    marginLeft: 4,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  statusRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 12,
  },
  statusInfo: {
    flex: 1,
  },
  statusLabel: {
    fontSize: 12,
    color: colors.textSecondary,
    marginBottom: 4,
  },
  statusBadge: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  statusDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    marginRight: 8,
  },
  statusText: {
    fontSize: 16,
    fontWeight: '500',
    color: colors.text,
  },
  actionButton: {
    marginTop: 12,
  },
  codeDisplay: {
    backgroundColor: colors.primary,
    borderRadius: 8,
    padding: 16,
    alignItems: 'center',
    marginBottom: 12,
  },
  codeLabel: {
    fontSize: 12,
    color: 'rgba(255,255,255,0.7)',
    marginBottom: 4,
  },
  code: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#ffffff',
    letterSpacing: 2,
    fontFamily: 'monospace',
  },
  codeInstructions: {
    fontSize: 12,
    color: colors.textSecondary,
    textAlign: 'center',
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
  settingRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  settingInfo: {
    flex: 1,
    marginRight: 16,
  },
  settingTitle: {
    fontSize: 16,
    fontWeight: '500',
    color: colors.text,
    marginBottom: 2,
  },
  settingDescription: {
    fontSize: 13,
    color: colors.textSecondary,
  },
  unavailableText: {
    fontSize: 11,
    color: colors.warning,
    marginTop: 4,
  },
  emailSection: {
    marginTop: 16,
    paddingTop: 16,
    borderTopWidth: 1,
    borderTopColor: colors.border,
  },
  saveButton: {
    marginTop: 12,
    alignSelf: 'flex-end',
  },
  pushCard: {
    marginTop: 12,
  },
  aboutRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingVertical: 8,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
  },
  aboutLabel: {
    fontSize: 14,
    color: colors.textSecondary,
  },
  aboutValue: {
    fontSize: 14,
    color: colors.text,
    fontWeight: '500',
  },
});
