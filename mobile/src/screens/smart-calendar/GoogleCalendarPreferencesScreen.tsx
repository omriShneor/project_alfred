import React, { useState, useEffect } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  Alert,
  Switch,
} from 'react-native';
import { Feather } from '@expo/vector-icons';
import { LoadingSpinner, Card, Button, Select } from '../../components/common';
import { colors } from '../../theme/colors';
import {
  useGCalStatus,
  useGCalSettings,
  useUpdateGCalSettings,
  useCalendars,
} from '../../hooks';
import type { Calendar } from '../../types/event';

export function GoogleCalendarPreferencesScreen() {
  const { data: gcalStatus, isLoading: statusLoading } = useGCalStatus();
  const { data: settings, isLoading: settingsLoading } = useGCalSettings();
  const { data: calendars, isLoading: calendarsLoading } = useCalendars(gcalStatus?.connected ?? false);
  const updateSettings = useUpdateGCalSettings();

  const [syncEnabled, setSyncEnabled] = useState<boolean>(false);
  const [selectedCalendarId, setSelectedCalendarId] = useState<string>('');
  const [hasChanges, setHasChanges] = useState(false);

  // Initialize from settings
  useEffect(() => {
    if (settings) {
      setSyncEnabled(settings.sync_enabled);
      if (!selectedCalendarId) {
        setSelectedCalendarId(settings.selected_calendar_id || 'primary');
      }
    }
  }, [settings, selectedCalendarId]);

  // Track changes
  useEffect(() => {
    if (settings) {
      const calendarChanged = selectedCalendarId !== settings.selected_calendar_id;
      const syncChanged = syncEnabled !== settings.sync_enabled;
      setHasChanges(calendarChanged || syncChanged);
    }
  }, [selectedCalendarId, syncEnabled, settings]);

  const handleSave = async () => {
    // Only require calendar selection if sync is enabled
    if (syncEnabled && !selectedCalendarId) {
      Alert.alert('Error', 'Please select a calendar');
      return;
    }

    const selectedCalendar = calendars?.find((c: Calendar) => c.id === selectedCalendarId);
    const calendarName = selectedCalendar?.summary || 'Primary';

    try {
      await updateSettings.mutateAsync({
        sync_enabled: syncEnabled,
        selected_calendar_id: selectedCalendarId || 'primary',
        selected_calendar_name: calendarName,
      });
      setHasChanges(false);
      Alert.alert('Success', 'Settings saved');
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to save settings');
    }
  };

  const isLoading = statusLoading || settingsLoading || calendarsLoading;
  const isConnected = gcalStatus?.connected ?? false;

  // Calendar dropdown options
  const calendarOptions = calendars?.map((c: Calendar) => ({
    label: c.summary + (c.primary ? ' (Primary)' : ''),
    value: c.id,
  })) || [];

  if (isLoading) {
    return (
      <View style={styles.screen}>
        <View style={styles.loadingContainer}>
          <LoadingSpinner />
        </View>
      </View>
    );
  }

  if (!isConnected) {
    return (
      <View style={styles.screen}>
        <ScrollView style={styles.container} contentContainerStyle={styles.content}>
          <Card>
            <View style={styles.emptyState}>
              <Feather name="calendar" size={48} color={colors.textSecondary} />
              <Text style={styles.emptyStateTitle}>Google Not Connected</Text>
              <Text style={styles.emptyStateText}>
                Connect your Google account first to configure calendar sync.
              </Text>
            </View>
          </Card>
        </ScrollView>
      </View>
    );
  }

  return (
    <View style={styles.screen}>
      <ScrollView style={styles.container} contentContainerStyle={styles.content}>
        {/* Current Status */}
        <Text style={styles.sectionTitle}>Current Status</Text>
        <Card>
          <View style={styles.statusItem}>
            <View style={styles.statusLeft}>
              <Feather
                name={syncEnabled ? 'check-circle' : 'x-circle'}
                size={20}
                color={syncEnabled ? colors.success : colors.textSecondary}
              />
              <Text style={styles.statusLabel}>
                {syncEnabled
                  ? `Syncing to: ${settings?.selected_calendar_name || 'Primary'}`
                  : 'Events stored locally only'}
              </Text>
            </View>
          </View>
        </Card>

        {/* Sync Toggle */}
        <Text style={styles.sectionTitle}>Sync Settings</Text>
        <Card>
          <View style={styles.toggleRow}>
            <View style={styles.toggleInfo}>
              <Text style={styles.toggleLabel}>Sync to Google Calendar</Text>
              <Text style={styles.toggleDescription}>
                When enabled, confirmed events will be added to your Google Calendar
              </Text>
            </View>
            <Switch
              value={syncEnabled}
              onValueChange={setSyncEnabled}
              trackColor={{ false: colors.border, true: colors.primary }}
              thumbColor="#ffffff"
            />
          </View>
        </Card>

        {/* Calendar Selection - only show when sync is enabled */}
        {syncEnabled && (
          <>
            <Text style={styles.sectionTitle}>Target Calendar</Text>
            <Card>
              <Text style={styles.helpText}>
                Events detected from all sources (WhatsApp, Telegram, Gmail) will sync to this calendar.
              </Text>
              <View style={styles.selectContainer}>
                <Select
                  options={calendarOptions}
                  value={selectedCalendarId}
                  onChange={(value) => setSelectedCalendarId(value)}
                  placeholder="Select a calendar"
                />
              </View>
            </Card>
          </>
        )}

        {/* Save Button */}
        {hasChanges && (
          <View style={styles.saveContainer}>
            <Button
              title="Save Changes"
              onPress={handleSave}
              loading={updateSettings.isPending}
            />
          </View>
        )}
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  screen: {
    flex: 1,
    backgroundColor: colors.background,
  },
  container: {
    flex: 1,
  },
  content: {
    padding: 16,
  },
  loadingContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  sectionTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.textSecondary,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
    marginBottom: 8,
    marginLeft: 4,
    marginTop: 16,
  },
  emptyState: {
    alignItems: 'center',
    paddingVertical: 32,
  },
  emptyStateTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: colors.text,
    marginTop: 16,
  },
  emptyStateText: {
    fontSize: 14,
    color: colors.textSecondary,
    marginTop: 8,
    textAlign: 'center',
    paddingHorizontal: 16,
  },
  statusItem: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 4,
  },
  statusLeft: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  statusLabel: {
    fontSize: 15,
    color: colors.text,
    marginLeft: 12,
  },
  statusValue: {
    fontSize: 15,
    fontWeight: '500',
    color: colors.primary,
  },
  helpText: {
    fontSize: 13,
    color: colors.textSecondary,
    marginBottom: 12,
    lineHeight: 18,
  },
  selectContainer: {
    marginTop: 4,
  },
  saveContainer: {
    marginTop: 24,
  },
  toggleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  toggleInfo: {
    flex: 1,
    marginRight: 16,
  },
  toggleLabel: {
    fontSize: 15,
    fontWeight: '500',
    color: colors.text,
  },
  toggleDescription: {
    fontSize: 13,
    color: colors.textSecondary,
    marginTop: 4,
  },
});
