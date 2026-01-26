import React, { useState, useEffect } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TextInput,
  Switch,
  Alert,
  Platform,
} from 'react-native';
import { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { Button, Card } from '../../components/common';
import { colors } from '../../theme/colors';
import { getNotificationPrefs, updateEmailPrefs } from '../../api';
import { usePushNotifications } from '../../hooks';

type OnboardingStackParamList = {
  Welcome: undefined;
  WhatsAppSetup: undefined;
  GoogleCalendarSetup: undefined;
  NotificationSetup: undefined;
};

interface Props {
  navigation: NativeStackNavigationProp<OnboardingStackParamList, 'NotificationSetup'>;
  onComplete: () => void;
}

export function NotificationSetupScreen({ navigation, onComplete }: Props) {
  const [emailEnabled, setEmailEnabled] = useState(false);
  const [emailAddress, setEmailAddress] = useState('');
  const [emailAvailable, setEmailAvailable] = useState(false);
  const [pushAvailable, setPushAvailable] = useState(false);
  const [saving, setSaving] = useState(false);

  const {
    expoPushToken,
    permissionStatus,
    isRegistering,
    error: pushError,
    requestPermissions,
  } = usePushNotifications();

  const isPushSetup = expoPushToken !== null && permissionStatus === 'granted';

  useEffect(() => {
    loadPreferences();
  }, []);

  const loadPreferences = async () => {
    try {
      const prefs = await getNotificationPrefs();
      setEmailEnabled(prefs.preferences.email_enabled);
      setEmailAddress(prefs.preferences.email_address || '');
      setEmailAvailable(prefs.available.email);
      setPushAvailable(prefs.available.push);
    } catch (error) {
      // Ignore errors, just use defaults
    }
  };

  const handleEnablePush = async () => {
    const success = await requestPermissions();
    if (!success && pushError) {
      Alert.alert('Push Notifications', pushError);
    }
  };

  const handleSaveAndComplete = async () => {
    if (emailEnabled && emailAddress.trim()) {
      setSaving(true);
      try {
        await updateEmailPrefs(emailEnabled, emailAddress.trim());
      } catch (error: any) {
        Alert.alert(
          'Warning',
          'Could not save notification preferences. You can update them later in Settings.'
        );
      }
      setSaving(false);
    }
    onComplete();
  };

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <View style={styles.header}>
        <Text style={styles.stepIndicator}>Step 3 of 3</Text>
        <Text style={styles.title}>Notification Settings</Text>
        <Text style={styles.subtitle}>
          Choose how you want to be notified about new events
        </Text>
      </View>

      <Card>
        <View style={styles.settingRow}>
          <View style={styles.settingInfo}>
            <Text style={styles.settingTitle}>Email Notifications</Text>
            <Text style={styles.settingDescription}>
              Get notified when new events are detected
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
          <View style={styles.emailInput}>
            <Text style={styles.inputLabel}>Email Address</Text>
            <TextInput
              style={styles.input}
              value={emailAddress}
              onChangeText={setEmailAddress}
              placeholder="your@email.com"
              keyboardType="email-address"
              autoCapitalize="none"
              autoComplete="email"
            />
          </View>
        )}
      </Card>

      <Card style={styles.infoCard}>
        <Text style={styles.infoTitle}>What you'll receive:</Text>
        <View style={styles.infoList}>
          <InfoItem text="Notifications when events are detected from messages" />
          <InfoItem text="Reminders to review pending events" />
          <InfoItem text="Confirmation when events are synced to calendar" />
        </View>
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
            </View>
            {isPushSetup && (
              <View style={styles.enabledBadge}>
                <Text style={styles.enabledBadgeText}>Enabled</Text>
              </View>
            )}
          </View>

          {!isPushSetup && (
            <View style={styles.pushButtonContainer}>
              {permissionStatus === 'denied' ? (
                <Text style={styles.permissionDeniedText}>
                  Permission denied. Enable in device settings.
                </Text>
              ) : (
                <Button
                  title="Enable Push Notifications"
                  onPress={handleEnablePush}
                  loading={isRegistering}
                  variant="outline"
                />
              )}
            </View>
          )}
        </Card>
      )}

      {/* Coming Soon - SMS */}
      <Card style={styles.comingSoonCard}>
        <Text style={styles.comingSoonTitle}>Coming Soon</Text>
        <Text style={styles.comingSoonText}>
          SMS alerts will be available in a future update.
        </Text>
      </Card>

      <View style={styles.footer}>
        <Button
          title="Complete Setup"
          onPress={handleSaveAndComplete}
          loading={saving}
          size="large"
          style={styles.completeButton}
        />
        <Button
          title="Skip Notifications"
          onPress={onComplete}
          variant="outline"
          style={styles.skipButton}
        />
      </View>
    </ScrollView>
  );
}

function InfoItem({ text }: { text: string }) {
  return (
    <View style={styles.infoItem}>
      <Text style={styles.infoBullet}>•</Text>
      <Text style={styles.infoText}>{text}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.background,
  },
  content: {
    padding: 24,
  },
  header: {
    marginBottom: 24,
  },
  stepIndicator: {
    fontSize: 12,
    color: colors.primary,
    fontWeight: '600',
    marginBottom: 8,
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
    color: colors.text,
    marginBottom: 8,
  },
  subtitle: {
    fontSize: 14,
    color: colors.textSecondary,
    lineHeight: 20,
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
    fontWeight: '600',
    color: colors.text,
    marginBottom: 4,
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
  emailInput: {
    marginTop: 16,
    paddingTop: 16,
    borderTopWidth: 1,
    borderTopColor: colors.border,
  },
  inputLabel: {
    fontSize: 14,
    fontWeight: '500',
    color: colors.text,
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
  infoCard: {
    marginTop: 16,
  },
  infoTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 12,
  },
  infoList: {
    gap: 8,
  },
  infoItem: {
    flexDirection: 'row',
    alignItems: 'flex-start',
  },
  infoBullet: {
    fontSize: 14,
    color: colors.primary,
    marginRight: 8,
    fontWeight: 'bold',
  },
  infoText: {
    flex: 1,
    fontSize: 13,
    color: colors.text,
    lineHeight: 18,
  },
  pushCard: {
    marginTop: 16,
  },
  pushButtonContainer: {
    marginTop: 12,
  },
  enabledBadge: {
    backgroundColor: colors.success,
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 12,
  },
  enabledBadgeText: {
    color: '#ffffff',
    fontSize: 12,
    fontWeight: '600',
  },
  permissionDeniedText: {
    fontSize: 13,
    color: colors.warning,
    textAlign: 'center',
  },
  comingSoonCard: {
    marginTop: 16,
    backgroundColor: '#f0f0f0',
  },
  comingSoonTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.textSecondary,
    marginBottom: 4,
  },
  comingSoonText: {
    fontSize: 13,
    color: colors.textSecondary,
  },
  footer: {
    marginTop: 32,
    gap: 12,
  },
  completeButton: {
    width: '100%',
  },
  skipButton: {
    width: '100%',
  },
});
