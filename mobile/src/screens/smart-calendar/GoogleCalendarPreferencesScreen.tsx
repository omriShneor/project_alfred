import React, { useState, useEffect, useRef } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  Alert,
  Switch,
} from 'react-native';
import { Feather } from '@expo/vector-icons';
import { LoadingSpinner, Card, Select } from '../../components/common';
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
  const initializedRef = useRef(false);

  // Initialize from settings (only once)
  useEffect(() => {
    if (settings && !initializedRef.current) {
      initializedRef.current = true;
      setSyncEnabled(settings.sync_enabled);
      if (settings.sync_enabled && settings.selected_calendar_id) {
        setSelectedCalendarId(settings.selected_calendar_id);
      }
    }
  }, [settings]);

  // Handle sync toggle change - auto save
  const handleSyncToggle = async (enabled: boolean) => {
    setSyncEnabled(enabled);

    // If disabling sync, save immediately
    if (!enabled) {
      try {
        await updateSettings.mutateAsync({
          sync_enabled: false,
          selected_calendar_id: selectedCalendarId,
          selected_calendar_name: settings?.selected_calendar_name || '',
        });
      } catch (error: any) {
        Alert.alert('Error', error.message || 'Failed to update settings');
        setSyncEnabled(!enabled); // Revert on error
      }
    }
    // If enabling sync, wait for calendar selection
  };

  // Handle calendar selection change - auto save
  const handleCalendarChange = async (calendarId: string) => {
    setSelectedCalendarId(calendarId);

    const selectedCalendar = calendars?.find((c: Calendar) => c.id === calendarId);
    const calendarName = selectedCalendar?.summary || '';

    try {
      await updateSettings.mutateAsync({
        sync_enabled: syncEnabled,
        selected_calendar_id: calendarId,
        selected_calendar_name: calendarName,
      });
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to update settings');
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
                name={syncEnabled && selectedCalendarId ? 'check-circle' : syncEnabled ? 'alert-circle' : 'calendar'}
                size={20}
                color={syncEnabled && selectedCalendarId ? colors.success : syncEnabled ? colors.warning : colors.primary}
              />
              <Text style={styles.statusLabel}>
                {syncEnabled
                  ? selectedCalendarId
                    ? `Syncing to: ${settings?.selected_calendar_name || 'Selected calendar'}`
                    : 'Select a calendar from the list below'
                  : "Events stored in Alfred's calendar only"}
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
              onValueChange={handleSyncToggle}
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
                  onChange={handleCalendarChange}
                  placeholder="Select a calendar"
                />
              </View>
            </Card>
          </>
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
  helpText: {
    fontSize: 13,
    color: colors.textSecondary,
    marginBottom: 12,
    lineHeight: 18,
  },
  selectContainer: {
    marginTop: 4,
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
