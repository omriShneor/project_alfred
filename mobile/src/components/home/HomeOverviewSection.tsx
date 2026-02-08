import React from 'react';
import { View, Text, StyleSheet, TouchableOpacity } from 'react-native';
import {
  useNavigation,
  type NavigationProp,
  type ParamListBase,
} from '@react-navigation/native';
import { Ionicons } from '@expo/vector-icons';
import { Card, LoadingSpinner } from '../common';
import { useAppStatus } from '../../hooks/useAppStatus';
import { useChannels } from '../../hooks/useChannels';
import { useEvents } from '../../hooks/useEvents';
import { useReminders } from '../../hooks/useReminders';
import { useTodayEvents } from '../../hooks/useTodayEvents';
import {
  useGCalStatus,
  useWhatsAppStatus,
} from '../../hooks/useOnboardingStatus';
import {
  useTelegramChannels,
  useTelegramStatus,
} from '../../hooks/useTelegram';
import { useEmailSources, useGmailStatus } from '../../hooks/useGmail';
import { useAuth } from '../../auth';
import { colors } from '../../theme/colors';
import type { TodayEvent } from '../../types/calendar';
import type { AppStatus, ConnectionStatus } from '../../types/app';

interface OverviewWarning {
  id: string;
  text: string;
  icon: keyof typeof Ionicons.glyphMap;
  color: string;
  onPress: () => void;
}

type ReauthSource = 'whatsapp' | 'telegram' | 'gmail' | 'google_calendar';

interface SourceHealthIssue {
  source: ReauthSource;
  label: string;
  reason: string;
}

function getGreetingHour(hour: number) {
  if (hour < 12) {
    return 'Good morning';
  }
  if (hour < 18) {
    return 'Good afternoon';
  }
  return 'Good evening';
}

function getFirstName(name?: string | null) {
  if (!name) {
    return null;
  }

  const trimmedName = name.trim();
  if (!trimmedName) {
    return null;
  }

  return trimmedName.split(/\s+/)[0];
}

function formatTodayDate() {
  const now = new Date();
  return now.toLocaleDateString(undefined, {
    weekday: 'long',
    month: 'long',
    day: 'numeric',
  });
}

function formatTime(dateString: string) {
  return new Date(dateString).toLocaleTimeString(undefined, {
    hour: 'numeric',
    minute: '2-digit',
  });
}

function getNextEvent(events: TodayEvent[]) {
  const now = Date.now();
  return (
    events.find((event) => {
      if (event.all_day) {
        return false;
      }
      return new Date(event.end_time).getTime() >= now;
    }) ?? null
  );
}

function getConnectedSources(status?: AppStatus) {
  const sourceStates: ConnectionStatus[] = [
    status?.whatsapp,
    status?.telegram,
    status?.gmail,
    status?.google_calendar,
  ].filter((source): source is ConnectionStatus => source !== undefined);

  const enabled = sourceStates.filter((source) => Boolean(source.enabled)).length;
  const connected = sourceStates.filter(
    (source) => Boolean(source.enabled) && Boolean(source.connected)
  ).length;

  return { enabled, connected };
}

function hasEnabledSource(items?: Array<{ enabled?: boolean }>) {
  return Boolean(items?.some((item) => item.enabled));
}

export function HomeOverviewSection() {
  const navigation = useNavigation<NavigationProp<ParamListBase>>();
  const { user } = useAuth();

  const { data: appStatus, isLoading: loadingAppStatus } = useAppStatus();
  const { data: whatsappStatus, isLoading: loadingWhatsAppStatus } =
    useWhatsAppStatus();
  const { data: telegramStatus, isLoading: loadingTelegramStatus } =
    useTelegramStatus();
  const { data: gmailStatus, isLoading: loadingGmailStatus } = useGmailStatus();
  const { data: gcalStatus, isLoading: loadingGCalStatus } = useGCalStatus();
  const { data: whatsappChannels, isLoading: loadingWhatsappChannels } =
    useChannels();
  const { data: telegramChannels, isLoading: loadingTelegramChannels } =
    useTelegramChannels();
  const { data: emailSources, isLoading: loadingEmailSources } =
    useEmailSources();
  const { data: pendingEvents, isLoading: loadingPendingEvents } = useEvents({
    status: 'pending',
  });
  const { data: pendingReminders, isLoading: loadingPendingReminders } =
    useReminders({ status: 'pending' });
  const { data: confirmedReminders, isLoading: loadingConfirmedReminders } =
    useReminders({ status: 'confirmed' });
  const { data: syncedReminders, isLoading: loadingSyncedReminders } =
    useReminders({ status: 'synced' });
  const { data: todayEvents, isLoading: loadingTodayEvents } = useTodayEvents();

  const pendingEventsCount = pendingEvents?.length ?? 0;
  const pendingRemindersCount = pendingReminders?.length ?? 0;
  const pendingTotal = pendingEventsCount + pendingRemindersCount;
  const activeRemindersTotal =
    (confirmedReminders?.length ?? 0) + (syncedReminders?.length ?? 0);
  const todayTotal = todayEvents?.length ?? 0;

  const initialLoading =
    loadingAppStatus &&
    loadingPendingEvents &&
    loadingPendingReminders &&
    loadingConfirmedReminders &&
    loadingSyncedReminders &&
    loadingTodayEvents &&
    !pendingEvents &&
    !pendingReminders &&
    !confirmedReminders &&
    !syncedReminders &&
    !todayEvents;

  const connectedSources = getConnectedSources(appStatus);
  const nextEvent = getNextEvent(todayEvents ?? []);

  const unconfiguredConnectedSources = React.useMemo(() => {
    const sources: string[] = [];

    if (
      whatsappStatus?.connected &&
      !loadingWhatsappChannels &&
      !hasEnabledSource(whatsappChannels)
    ) {
      sources.push('WhatsApp');
    }

    if (
      telegramStatus?.connected &&
      !loadingTelegramChannels &&
      !hasEnabledSource(telegramChannels)
    ) {
      sources.push('Telegram');
    }

    const gmailConnected = gmailStatus?.connected || gmailStatus?.has_scopes;
    if (
      gmailConnected &&
      !loadingEmailSources &&
      !hasEnabledSource(emailSources)
    ) {
      sources.push('Gmail');
    }

    return sources;
  }, [
    whatsappStatus?.connected,
    loadingWhatsappChannels,
    whatsappChannels,
    telegramStatus?.connected,
    loadingTelegramChannels,
    telegramChannels,
    gmailStatus?.connected,
    gmailStatus?.has_scopes,
    loadingEmailSources,
    emailSources,
  ]);

  const sourceAuthIssues = React.useMemo<SourceHealthIssue[]>(() => {
    const issues: SourceHealthIssue[] = [];

    if (
      appStatus?.whatsapp?.enabled &&
      !loadingWhatsAppStatus &&
      !whatsappStatus?.connected
    ) {
      issues.push({
        source: 'whatsapp',
        label: 'WhatsApp',
        reason:
          whatsappStatus?.message ??
          'Session is no longer authenticated. Reconnect to resume tracking.',
      });
    }

    if (
      appStatus?.telegram?.enabled &&
      !loadingTelegramStatus &&
      !telegramStatus?.connected
    ) {
      issues.push({
        source: 'telegram',
        label: 'Telegram',
        reason:
          telegramStatus?.message ??
          'Session is no longer authenticated. Reconnect to resume tracking.',
      });
    }

    const gmailReady = Boolean(gmailStatus?.connected || gmailStatus?.has_scopes);
    if (appStatus?.gmail?.enabled && !loadingGmailStatus && !gmailReady) {
      issues.push({
        source: 'gmail',
        label: 'Gmail',
        reason:
          gmailStatus?.message ??
          'Gmail authorization is missing. Reconnect to keep scanning emails.',
      });
    }

    const gcalReady = Boolean(gcalStatus?.connected && gcalStatus?.has_scopes);
    if (
      appStatus?.google_calendar?.enabled &&
      !loadingGCalStatus &&
      !gcalReady
    ) {
      issues.push({
        source: 'google_calendar',
        label: 'Google Calendar',
        reason:
          gcalStatus?.message ??
          'Calendar authorization is missing. Reconnect to restore calendar sync.',
      });
    }

    return issues;
  }, [
    appStatus?.whatsapp?.enabled,
    appStatus?.telegram?.enabled,
    appStatus?.gmail?.enabled,
    appStatus?.google_calendar?.enabled,
    loadingWhatsAppStatus,
    loadingTelegramStatus,
    loadingGmailStatus,
    loadingGCalStatus,
    whatsappStatus?.connected,
    whatsappStatus?.message,
    telegramStatus?.connected,
    telegramStatus?.message,
    gmailStatus?.connected,
    gmailStatus?.has_scopes,
    gmailStatus?.message,
    gcalStatus?.connected,
    gcalStatus?.has_scopes,
    gcalStatus?.message,
  ]);

  const warningItems = React.useMemo<OverviewWarning[]>(() => {
    const warnings: OverviewWarning[] = [];

    if (sourceAuthIssues.length > 0) {
      const sourceList = sourceAuthIssues.map((issue) => issue.label).join(', ');
      warnings.push({
        id: 'source-auth-issues',
        text:
          sourceAuthIssues.length === 1
            ? `${sourceList} needs to be reconnected. Tap to fix authentication.`
            : `${sourceList} need to be reconnected. Tap to fix authentication.`,
        icon: 'warning-outline',
        color: colors.danger,
        onPress: () =>
          navigation.navigate('Preferences', {
            reauthSources: sourceAuthIssues.map((issue) => issue.source),
          }),
      });
    }

    if (unconfiguredConnectedSources.length > 0) {
      const sourceList = unconfiguredConnectedSources.join(', ');
      warnings.push({
        id: 'unconfigured-connected-sources',
        text:
          unconfiguredConnectedSources.length === 1
            ? `${sourceList} is connected, but no contacts/senders are being tracked yet.`
            : `${sourceList} are connected, but no contacts/senders are being tracked yet.`,
        icon: 'warning-outline',
        color: colors.warning,
        onPress: () => navigation.navigate('Preferences'),
      });
    }

    if (connectedSources.enabled === 0 && unconfiguredConnectedSources.length === 0) {
      warnings.push({
        id: 'no-enabled-sources',
        text: 'Connect at least one app so Alfred can start detecting events and reminders.',
        icon: 'link-outline',
        color: colors.primary,
        onPress: () => navigation.navigate('Preferences'),
      });
    }

    return warnings;
  }, [
    navigation,
    sourceAuthIssues,
    unconfiguredConnectedSources,
    connectedSources.enabled,
  ]);

  let infoText = 'You are caught up.';
  let infoIcon: keyof typeof Ionicons.glyphMap = 'checkmark-circle-outline';
  let infoColor = colors.success;

  if (pendingTotal > 0) {
    infoText = `You have ${pendingTotal} pending item${
      pendingTotal === 1 ? '' : 's'
    } to review.`;
    infoIcon = 'warning-outline';
    infoColor = colors.warning;
  } else if (nextEvent) {
    infoText = `Next event: ${nextEvent.summary} at ${formatTime(nextEvent.start_time)}.`;
    infoIcon = 'time-outline';
    infoColor = colors.primary;
  }

  const now = new Date();
  const greeting = getGreetingHour(now.getHours());
  const firstName = getFirstName(user?.name);
  const greetingText = firstName ? `${greeting}, ${firstName}` : greeting;

  if (initialLoading) {
    return (
      <View style={styles.container}>
        <Card style={styles.card}>
          <Text style={styles.sectionLabel}>TODAY OVERVIEW</Text>
          <View style={styles.loadingRow}>
            <LoadingSpinner size="small" />
            <Text style={styles.loadingText}>Preparing your dashboard...</Text>
          </View>
        </Card>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <Card style={styles.card}>
        <View style={styles.headerRow}>
          <View style={styles.headerText}>
            <Text style={styles.sectionLabel}>TODAY OVERVIEW</Text>
            <Text style={styles.title}>{greetingText}</Text>
            <Text style={styles.subtitle}>{formatTodayDate()}</Text>
          </View>
        </View>

        <View style={styles.statsRow}>
          <TouchableOpacity
            activeOpacity={0.75}
            style={[styles.statItem, styles.reviewStatItem]}
            onPress={() => navigation.navigate('NeedsReview')}
          >
            <View style={styles.reviewStatHeader}>
              <Text style={styles.statValue}>{pendingTotal}</Text>
              <View style={styles.reviewHeaderRight}>
                {pendingTotal > 0 ? <View style={styles.reviewDot} /> : null}
                <Ionicons
                  name="chevron-forward"
                  size={14}
                  color={colors.textSecondary}
                />
              </View>
            </View>
            <Text style={styles.statLabel}>Needs review</Text>
            <Text style={styles.reviewStatHint}>Tap to review</Text>
          </TouchableOpacity>
          <View style={styles.statItem}>
            <Text style={styles.statValue}>{todayTotal}</Text>
            <Text style={styles.statLabel}>Today events</Text>
          </View>
          <View style={styles.statItem}>
            <Text style={styles.statValue}>{activeRemindersTotal}</Text>
            <Text style={styles.statLabel}>Active reminders</Text>
          </View>
        </View>

        {warningItems.length > 0 ? (
          <View style={styles.warningList}>
            {warningItems.map((warning) => (
              <TouchableOpacity
                key={warning.id}
                activeOpacity={0.75}
                style={[styles.focusRow, { borderColor: warning.color + '40' }]}
                onPress={warning.onPress}
              >
                <Ionicons name={warning.icon} size={16} color={warning.color} />
                <Text style={styles.focusText}>{warning.text}</Text>
                <Ionicons
                  name="chevron-forward"
                  size={16}
                  color={colors.textSecondary}
                />
              </TouchableOpacity>
            ))}
          </View>
        ) : (
          <View style={[styles.focusRow, styles.infoRow, { borderColor: infoColor + '40' }]}>
            <Ionicons name={infoIcon} size={16} color={infoColor} />
            <Text style={styles.focusText}>{infoText}</Text>
          </View>
        )}

        <View style={styles.actionsRow}>
          <TouchableOpacity
            activeOpacity={0.75}
            style={styles.actionButton}
            onPress={() => navigation.navigate('Preferences')}
          >
            <Ionicons
              name="options-outline"
              size={16}
              color={colors.primary}
              style={styles.actionIcon}
            />
            <Text style={styles.actionText}>Manage connections</Text>
          </TouchableOpacity>
          <TouchableOpacity
            activeOpacity={0.75}
            style={styles.actionButton}
            onPress={() => navigation.navigate('Settings')}
          >
            <Ionicons
              name="settings-outline"
              size={16}
              color={colors.primary}
              style={styles.actionIcon}
            />
            <Text style={styles.actionText}>Notifications</Text>
          </TouchableOpacity>
        </View>
      </Card>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    marginBottom: 16,
  },
  card: {
    borderRadius: 14,
    borderWidth: 1,
    borderColor: colors.primary + '1f',
    backgroundColor: '#f7fbff',
    paddingVertical: 14,
    paddingHorizontal: 14,
  },
  sectionLabel: {
    fontSize: 11,
    fontWeight: '700',
    color: colors.primary,
    letterSpacing: 0.6,
    marginBottom: 2,
    textTransform: 'uppercase',
  },
  headerRow: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 12,
  },
  headerText: {
    flex: 1,
  },
  title: {
    fontSize: 22,
    fontWeight: '700',
    color: colors.text,
    marginBottom: 2,
  },
  subtitle: {
    fontSize: 13,
    color: colors.textSecondary,
  },
  statsRow: {
    flexDirection: 'row',
    marginBottom: 12,
    gap: 8,
  },
  statItem: {
    flex: 1,
    borderRadius: 10,
    backgroundColor: colors.card,
    paddingVertical: 10,
    paddingHorizontal: 8,
    borderWidth: 1,
    borderColor: colors.border,
  },
  reviewStatItem: {
    alignItems: 'flex-start',
  },
  reviewStatHeader: {
    width: '100%',
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 2,
  },
  reviewHeaderRight: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
  },
  reviewDot: {
    width: 12,
    height: 12,
    borderRadius: 5,
    backgroundColor: '#FD1D1D'//colors.danger,
  },
  statValue: {
    fontSize: 20,
    fontWeight: '700',
    color: colors.text,
    marginBottom: 2,
  },
  statLabel: {
    fontSize: 12,
    color: colors.textSecondary,
  },
  reviewStatHint: {
    fontSize: 11,
    color: colors.textSecondary,
    marginTop: 2,
  },
  warningList: {
    marginBottom: 10,
    gap: 8,
  },
  focusRow: {
    flexDirection: 'row',
    alignItems: 'center',
    borderWidth: 1,
    backgroundColor: colors.card,
    borderRadius: 10,
    paddingVertical: 10,
    paddingHorizontal: 10,
  },
  infoRow: {
    marginBottom: 10,
  },
  focusText: {
    fontSize: 13,
    color: colors.text,
    marginLeft: 8,
    flex: 1,
  },
  actionsRow: {
    flexDirection: 'row',
    gap: 8,
  },
  actionButton: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: colors.card,
    borderRadius: 10,
    borderWidth: 1,
    borderColor: colors.border,
    paddingVertical: 10,
    paddingHorizontal: 8,
  },
  actionIcon: {
    marginRight: 6,
  },
  actionText: {
    fontSize: 13,
    color: colors.text,
    fontWeight: '600',
  },
  loadingRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingTop: 10,
  },
  loadingText: {
    marginLeft: 10,
    fontSize: 13,
    color: colors.textSecondary,
  },
});
