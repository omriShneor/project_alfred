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
import { SafeAreaView } from 'react-native-safe-area-context';
import { Button, Card, LoadingSpinner } from '../components/common';
import { colors } from '../theme/colors';
import { usePushNotifications } from '../hooks';
import { getNotificationPrefs, updateEmailPrefs, updatePushPrefs } from '../api';

export function SettingsScreen() {
  const [emailEnabled, setEmailEnabled] = useState(false);
  const [emailAddress, setEmailAddress] = useState('');
  const [emailAvailable, setEmailAvailable] = useState(false);
  const [pushEnabled, setPushEnabled] = useState(false);
  const [pushAvailable, setPushAvailable] = useState(false);
  const [savingNotifications, setSavingNotifications] = useState(false);

  // Track original values to detect changes
  const [originalEmailEnabled, setOriginalEmailEnabled] = useState(false);
  const [originalEmailAddress, setOriginalEmailAddress] = useState('');
  const [originalPushEnabled, setOriginalPushEnabled] = useState(false);

  const {
    expoPushToken,
    permissionStatus,
    isRegistering,
    requestPermissions,
  } = usePushNotifications();

  useEffect(() => {
    loadNotificationPrefs();
  }, []);

  const loadNotificationPrefs = async () => {
    try {
      const prefs = await getNotificationPrefs();
      setEmailEnabled(prefs.preferences.email_enabled);
      setEmailAddress(prefs.preferences.email_address || '');
      setEmailAvailable(prefs.available.email);
      setPushEnabled(prefs.preferences.push_enabled);
      setPushAvailable(prefs.available.push);
      // Store original values
      setOriginalEmailEnabled(prefs.preferences.email_enabled);
      setOriginalEmailAddress(prefs.preferences.email_address || '');
      setOriginalPushEnabled(prefs.preferences.push_enabled);
    } catch (error) {
      // Ignore errors
    }
  };

  const hasNotificationChanges =
    emailEnabled !== originalEmailEnabled ||
    emailAddress !== originalEmailAddress ||
    pushEnabled !== originalPushEnabled;

  const handleSaveNotificationSettings = async () => {
    if (emailEnabled && !emailAddress.trim()) {
      Alert.alert('Error', 'Please enter an email address');
      return;
    }

    setSavingNotifications(true);
    try {
      await updateEmailPrefs(emailEnabled, emailAddress.trim());

      if (pushAvailable) {
        if (pushEnabled && !expoPushToken && originalPushEnabled !== pushEnabled) {
          if (permissionStatus === 'denied') {
            Alert.alert(
              'Permission Required',
              'Push notifications are disabled. Please enable them in your device settings.'
            );
            setSavingNotifications(false);
            return;
          }
          const success = await requestPermissions();
          if (!success) {
            setSavingNotifications(false);
            return;
          }
        } else if (pushEnabled !== originalPushEnabled) {
          await updatePushPrefs(pushEnabled);
        }
      }

      setOriginalEmailEnabled(emailEnabled);
      setOriginalEmailAddress(emailAddress);
      setOriginalPushEnabled(pushEnabled);

      Alert.alert('Success', 'Notification settings saved');
    } catch (error: any) {
      Alert.alert('Error', 'Failed to save notification settings');
    }
    setSavingNotifications(false);
  };

  return (
    <SafeAreaView style={styles.container} edges={['top']}>
      <ScrollView style={styles.scrollView} contentContainerStyle={styles.content}>
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
            </View>
          )}

          {Platform.OS !== 'web' && (
            <View style={styles.pushSection}>
              <View style={styles.settingRow}>
                <View style={styles.settingInfo}>
                  <Text style={styles.settingTitle}>Push Notifications</Text>
                  <Text style={styles.settingDescription}>
                    Get instant alerts on your phone
                  </Text>
                  {!pushAvailable && (
                    <Text style={styles.unavailableText}>
                      Push notifications not configured on server
                    </Text>
                  )}
                  {permissionStatus === 'denied' && (
                    <Text style={styles.unavailableText}>
                      Permission denied - enable in device settings
                    </Text>
                  )}
                </View>
                {isRegistering ? (
                  <LoadingSpinner size="small" />
                ) : (
                  <Switch
                    value={pushEnabled}
                    onValueChange={setPushEnabled}
                    disabled={!pushAvailable}
                    trackColor={{ false: colors.border, true: colors.primary }}
                    thumbColor="#ffffff"
                  />
                )}
              </View>
            </View>
          )}

          {hasNotificationChanges && (
            <Button
              title="Save Notification Settings"
              onPress={handleSaveNotificationSettings}
              loading={savingNotifications}
              style={styles.saveButton}
            />
          )}
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
  scrollView: {
    flex: 1,
  },
  content: {
    padding: 16,
    paddingTop: 16,
    paddingBottom: 32,
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
  input: {
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 8,
    padding: 12,
    fontSize: 16,
    color: colors.text,
    backgroundColor: colors.background,
  },
  emailSection: {
    marginTop: 16,
    paddingTop: 16,
    borderTopWidth: 1,
    borderTopColor: colors.border,
  },
  pushSection: {
    marginTop: 16,
    paddingTop: 16,
    borderTopWidth: 1,
    borderTopColor: colors.border,
  },
  saveButton: {
    marginTop: 20,
  },
});
