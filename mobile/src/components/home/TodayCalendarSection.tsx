import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { LoadingSpinner } from '../common';
import { useTodayEvents } from '../../hooks/useTodayEvents';
import { colors } from '../../theme/colors';
import type { TodayEvent } from '../../types/calendar';

type TimingBadge = 'now' | 'next' | null;

function getTimingBadge(event: TodayEvent): TimingBadge {
  if (event.all_day) {
    return null;
  }

  const now = Date.now();
  const start = new Date(event.start_time).getTime();
  const end = new Date(event.end_time).getTime();
  if (Number.isNaN(start) || Number.isNaN(end)) {
    return null;
  }

  if (now >= start && now <= end) {
    return 'now';
  }

  if (start > now && start - now <= 60 * 60 * 1000) {
    return 'next';
  }

  return null;
}

function formatTime(dateString: string) {
  const date = new Date(dateString);
  if (Number.isNaN(date.getTime())) {
    return '--';
  }
  return date.toLocaleTimeString(undefined, {
    hour: 'numeric',
    minute: '2-digit',
  });
}

function formatTimeRange(start: string, end: string) {
  return `${formatTime(start)} - ${formatTime(end)}`;
}

function getSourceLabel(source?: TodayEvent['source']) {
  if (source === 'google') {
    return 'Google';
  }
  if (source === 'outlook') {
    return 'Outlook';
  }
  return 'Alfred';
}

function TimelineEvent({ event, isLast }: { event: TodayEvent; isLast: boolean }) {
  const timingBadge = getTimingBadge(event);
  const eventTitle = event.summary?.trim() || 'Untitled event';
  const sourceLabel = getSourceLabel(event.source);
  const sourceDotStyle =
    event.source === 'google'
      ? styles.sourceDotGoogle
      : event.source === 'outlook'
        ? styles.sourceDotOutlook
        : styles.sourceDotAlfred;

  return (
    <View style={[styles.timelineItem, isLast && styles.timelineItemLast]}>
      <View style={styles.timePanel}>
        <Text style={styles.timePrimary}>
          {event.all_day ? 'All day' : formatTime(event.start_time)}
        </Text>
        <Text style={styles.timeSecondary}>
          {event.all_day ? 'Today' : formatTime(event.end_time)}
        </Text>
      </View>
      <View style={styles.eventDetails}>
        <View style={styles.titleRow}>
          <Text style={styles.eventTitle} numberOfLines={1}>
            {eventTitle}
          </Text>
          {timingBadge && (
            <View
              style={[
                styles.timingBadge,
                timingBadge === 'now'
                  ? styles.timingBadgeNow
                  : styles.timingBadgeNext,
              ]}
            >
              <Text
                style={[
                  styles.timingBadgeText,
                  timingBadge === 'now'
                    ? styles.timingBadgeTextNow
                    : styles.timingBadgeTextNext,
                ]}
              >
                {timingBadge === 'now' ? 'Now' : 'Up next'}
              </Text>
            </View>
          )}
        </View>
        {!event.all_day && (
          <Text style={styles.eventTimeRange}>
            {formatTimeRange(event.start_time, event.end_time)}
          </Text>
        )}
        <View style={styles.metaRow}>
          <View style={styles.sourceChip}>
            <View style={[styles.sourceDot, sourceDotStyle]} />
            <Text style={styles.sourceText}>{sourceLabel}</Text>
          </View>
          {event.location ? (
            <View style={styles.locationRow}>
              <Ionicons
                name="location-outline"
                size={12}
                color={colors.textSecondary}
              />
              <Text style={styles.eventLocation} numberOfLines={1}>
                {event.location}
              </Text>
            </View>
          ) : null}
        </View>
      </View>
    </View>
  );
}

export function TodayCalendarSection() {
  const { data: events, isLoading, isError } = useTodayEvents();
  const sortedEvents = React.useMemo(() => {
    if (!events) {
      return [];
    }

    return [...events].sort(
      (a, b) =>
        new Date(a.start_time).getTime() - new Date(b.start_time).getTime()
    );
  }, [events]);

  const scheduleSummary = React.useMemo(() => {
    if (sortedEvents.length === 0) {
      return '';
    }

    const now = Date.now();
    const allDayCount = sortedEvents.filter((event) => event.all_day).length;
    const upcomingCount = sortedEvents.filter((event) => {
      if (event.all_day) {
        return true;
      }
      return new Date(event.end_time).getTime() >= now;
    }).length;

    if (allDayCount > 0 && upcomingCount > 0) {
      return `${upcomingCount} upcoming • ${allDayCount} all day`;
    }
    if (upcomingCount > 0) {
      return `${upcomingCount} upcoming`;
    }
    if (allDayCount > 0) {
      return `${allDayCount} all day`;
    }

    return 'No upcoming events';
  }, [sortedEvents]);

  if (isLoading) {
    return (
      <View style={styles.container}>
        <Text style={styles.sectionTitle}>TODAY'S SCHEDULE</Text>
        <View style={styles.stateCard}>
          <LoadingSpinner size="small" />
          <Text style={styles.stateSubtext}>Loading today&apos;s events...</Text>
        </View>
      </View>
    );
  }

  if (isError) {
    return (
      <View style={styles.container}>
        <Text style={styles.sectionTitle}>TODAY'S SCHEDULE</Text>
        <View style={[styles.stateCard, styles.errorState]}>
          <Ionicons
            name="cloud-offline-outline"
            size={20}
            color={colors.danger}
            style={styles.stateIcon}
          />
          <Text style={styles.errorText}>Unable to load calendar</Text>
          <Text style={styles.errorSubtext}>
            Please try again later
          </Text>
        </View>
      </View>
    );
  }

  if (!events || events.length === 0) {
    return (
      <View style={styles.container}>
        <Text style={styles.sectionTitle}>TODAY'S SCHEDULE</Text>
        <View style={styles.stateCard}>
          <Ionicons
            name="calendar-outline"
            size={20}
            color={colors.primary}
            style={styles.stateIcon}
          />
          <Text style={styles.emptyText}>No events scheduled for today</Text>
          <Text style={styles.emptySubtext}>
            Events will appear here once confirmed
          </Text>
        </View>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <Text style={styles.sectionTitle}>
        TODAY'S SCHEDULE ({sortedEvents.length})
      </Text>
      <View style={styles.card}>
        <View style={styles.summaryRow}>
          <View style={styles.summaryItem}>
            <Ionicons name="time-outline" size={14} color={colors.primary} />
            <Text style={styles.summaryText}>{scheduleSummary}</Text>
          </View>
        </View>
        {sortedEvents.map((event, index) => (
          <TimelineEvent
            key={event.id}
            event={event}
            isLast={index === sortedEvents.length - 1}
          />
        ))}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    marginBottom: 16,
  },
  sectionTitle: {
    fontSize: 12,
    fontWeight: '700',
    color: colors.textSecondary,
    marginBottom: 8,
    marginLeft: 4,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  card: {
    borderRadius: 14,
    borderWidth: 1,
    borderColor: colors.primary + '1f',
    backgroundColor: '#f7fbff',
    overflow: 'hidden',
  },
  summaryRow: {
    paddingHorizontal: 12,
    paddingVertical: 10,
    borderBottomWidth: 1,
    borderBottomColor: colors.primary + '20',
    backgroundColor: colors.card + 'd0',
  },
  summaryItem: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  summaryText: {
    fontSize: 12,
    color: colors.text,
    marginLeft: 6,
    fontWeight: '600',
  },
  timelineItem: {
    flexDirection: 'row',
    paddingVertical: 12,
    paddingHorizontal: 12,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
  },
  timelineItemLast: {
    borderBottomWidth: 0,
  },
  timePanel: {
    width: 84,
    paddingRight: 10,
    alignItems: 'flex-start',
  },
  timePrimary: {
    fontSize: 13,
    fontWeight: '700',
    color: colors.text,
  },
  timeSecondary: {
    fontSize: 11,
    color: colors.textSecondary,
    marginTop: 2,
  },
  eventDetails: {
    flex: 1,
  },
  titleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 2,
  },
  eventTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.text,
    flex: 1,
    marginRight: 6,
  },
  timingBadge: {
    borderRadius: 999,
    borderWidth: 1,
    paddingVertical: 2,
    paddingHorizontal: 6,
  },
  timingBadgeNow: {
    borderColor: colors.success + '60',
    backgroundColor: colors.success + '12',
  },
  timingBadgeNext: {
    borderColor: colors.primary + '60',
    backgroundColor: colors.primary + '12',
  },
  timingBadgeText: {
    fontSize: 10,
    fontWeight: '700',
    textTransform: 'uppercase',
    letterSpacing: 0.3,
  },
  timingBadgeTextNow: {
    color: colors.success,
  },
  timingBadgeTextNext: {
    color: colors.primary,
  },
  eventTimeRange: {
    fontSize: 12,
    color: colors.textSecondary,
    marginBottom: 6,
  },
  metaRow: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  sourceChip: {
    flexDirection: 'row',
    alignItems: 'center',
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 999,
    paddingVertical: 2,
    paddingHorizontal: 8,
    backgroundColor: colors.card,
  },
  sourceDot: {
    width: 6,
    height: 6,
    borderRadius: 3,
    marginRight: 6,
  },
  sourceDotGoogle: {
    backgroundColor: colors.primary,
  },
  sourceDotOutlook: {
    backgroundColor: colors.warning,
  },
  sourceDotAlfred: {
    backgroundColor: colors.success,
  },
  sourceText: {
    fontSize: 11,
    fontWeight: '600',
    color: colors.textSecondary,
  },
  locationRow: {
    marginLeft: 8,
    flexDirection: 'row',
    alignItems: 'center',
    flex: 1,
  },
  eventLocation: {
    fontSize: 12,
    color: colors.textSecondary,
    marginLeft: 4,
    flex: 1,
  },
  stateCard: {
    backgroundColor: colors.card,
    borderRadius: 14,
    borderWidth: 1,
    borderColor: colors.primary + '1f',
    padding: 18,
    alignItems: 'center',
  },
  stateIcon: {
    marginBottom: 6,
  },
  emptyText: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 4,
  },
  emptySubtext: {
    fontSize: 12,
    color: colors.textSecondary,
    textAlign: 'center',
  },
  stateSubtext: {
    marginTop: 8,
    fontSize: 12,
    color: colors.textSecondary,
  },
  errorState: {
    borderColor: colors.danger + '30',
  },
  errorText: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.danger,
    marginBottom: 4,
  },
  errorSubtext: {
    fontSize: 12,
    color: colors.textSecondary,
    textAlign: 'center',
  },
});
