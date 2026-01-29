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
import { useNavigation } from '@react-navigation/native';
import { Button, Card, LoadingSpinner } from '../components/common';
import { colors } from '../theme/colors';
import {
  useFeatures,
  useUpdateSmartCalendar,
  usePushNotifications,
  useWhatsAppStatus,
  useGCalStatus,
} from '../hooks';
import { getNotificationPrefs, updateEmailPrefs, updatePushPrefs } from '../api';
import type { DrawerNavigationProp } from '@react-navigation/drawer';
import type { DrawerParamList } from '../navigation/DrawerNavigator';

type NavigationProp = DrawerNavigationProp<DrawerParamList>;

export function CapabilitiesScreen() {
  const navigation = useNavigation<NavigationProp>();
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

  const { data: features, isLoading: featuresLoading } = useFeatures();
  const updateSmartCalendar = useUpdateSmartCalendar();
  const { data: waStatus } = useWhatsAppStatus();
  const { data: gcalStatus } = useGCalStatus();

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

  // Detect if there are unsaved notification changes
  const hasNotificationChanges =
    emailEnabled !== originalEmailEnabled ||
    emailAddress !== originalEmailAddress ||
    pushEnabled !== originalPushEnabled;

  const handleToggleSmartCalendar = async (enabled: boolean) => {
    if (enabled) {
      // Check if all required integrations are already connected
      const inputs = features?.smart_calendar?.inputs;
      const calendars = features?.smart_calendar?.calendars;

      const needsWhatsApp = inputs?.whatsapp?.enabled ?? false;
      const needsGmail = inputs?.email?.enabled ?? false;
      const needsGoogleCalendar = calendars?.google_calendar?.enabled ?? false;

      // Google account is needed for either Gmail or Google Calendar
      const needsGoogle = needsGmail || needsGoogleCalendar;

      // Check if any inputs have been configured
      const hasAnyInputConfigured = needsWhatsApp || needsGmail;

      // If no inputs are configured yet, always go through setup flow
      if (!hasAnyInputConfigured) {
        navigation.navigate('SmartCalendarStack' as any, { screen: 'Setup' });
        return;
      }

      const whatsAppConnected = waStatus?.connected ?? false;
      const googleConnected = gcalStatus?.connected ?? false;

      // Check if all required integrations are already connected
      const allConnected =
        (!needsWhatsApp || whatsAppConnected) &&
        (!needsGoogle || googleConnected);

      if (allConnected) {
        // All integrations are already connected, enable directly and go to Home
        try {
          await updateSmartCalendar.mutateAsync({ enabled: true, setup_complete: true });
          navigation.navigate('Home');
        } catch (error: any) {
          Alert.alert('Error', error.message || 'Failed to enable Smart Calendar');
        }
      } else {
        // Some integrations need to be connected, go to permissions screen
        navigation.navigate('SmartCalendarStack' as any, { screen: 'Permissions' });
      }
    } else {
      // Disable Smart Calendar - clear both enabled and setup_complete
      try {
        await updateSmartCalendar.mutateAsync({ enabled: false, setup_complete: false });
      } catch (error: any) {
        Alert.alert('Error', error.message || 'Failed to disable Smart Calendar');
      }
    }
  };

  const handleSaveNotificationSettings = async () => {
    if (emailEnabled && !emailAddress.trim()) {
      Alert.alert('Error', 'Please enter an email address');
      return;
    }

    setSavingNotifications(true);
    try {
      // Save email settings
      await updateEmailPrefs(emailEnabled, emailAddress.trim());

      // Save push settings (only if push is available)
      if (pushAvailable) {
        // If enabling push and no token, request permissions first
        if (pushEnabled && !expoPushToken && originalPushEnabled !== pushEnabled) {
          if (permissionStatus === 'denied') {
            Alert.alert(
              'Permission Required',
              'Push notifications are disabled. Please enable them in your device settings.',
              [{ text: 'OK' }]
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

      // Update originals after successful save
      setOriginalEmailEnabled(emailEnabled);
      setOriginalEmailAddress(emailAddress);
      setOriginalPushEnabled(pushEnabled);

      Alert.alert('Success', 'Notification settings saved');
    } catch (error: any) {
      Alert.alert('Error', 'Failed to save notification settings');
    }
    setSavingNotifications(false);
  };

  const smartCalendarEnabled = features?.smart_calendar?.enabled ?? false;
  const smartCalendarSetupComplete = features?.smart_calendar?.setup_complete ?? false;

  // Toggle should only be ON when both enabled AND setup is complete
  const smartCalendarToggleValue = smartCalendarEnabled && smartCalendarSetupComplete;

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
        {/* Features Section */}
        <Text style={styles.sectionTitle}>Features</Text>
        <Card>
          <View style={styles.featureRow}>
            <View style={styles.featureInfo}>
              <Text style={styles.featureTitle}>Smart Calendar</Text>
              <Text style={styles.featureDescription}>
                Automatically detect events from your messages and sync to calendar
              </Text>
            </View>
            {featuresLoading ? (
              <LoadingSpinner size="small" />
            ) : (
              <Switch
                value={smartCalendarToggleValue}
                onValueChange={handleToggleSmartCalendar}
                disabled={updateSmartCalendar.isPending}
                trackColor={{ false: colors.border, true: colors.primary }}
                thumbColor="#ffffff"
              />
            )}
          </View>
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
            </View>
          )}

          {/* Push Notifications */}
          {Platform.OS !== 'web' && pushAvailable && (
            <View style={styles.pushSection}>
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

          {/* Unified Save Button */}
          {hasNotificationChanges && (
            <Button
              title="Save Notification Settings"
              onPress={handleSaveNotificationSettings}
              loading={savingNotifications}
              style={styles.notificationSaveButton}
            />
          )}
        </Card>

        {/* About Section */}
        <Text style={styles.sectionTitle}>About</Text>
        <Card>
          <View style={styles.aboutRow}>
            <Text style={styles.aboutLabel}>Version</Text>
            <Text style={styles.aboutValue}>1.0.0</Text>
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
  featureRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  featureInfo: {
    flex: 1,
    marginRight: 16,
  },
  featureTitle: {
    fontSize: 16,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 4,
  },
  featureDescription: {
    fontSize: 13,
    color: colors.textSecondary,
    lineHeight: 18,
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
  notificationSaveButton: {
    marginTop: 20,
  },
  aboutRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingVertical: 8,
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
