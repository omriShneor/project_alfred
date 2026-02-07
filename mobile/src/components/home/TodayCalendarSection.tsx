import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
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

  if (now >= start && now <= end) {
    return 'now';
  }

  if (start > now && start - now <= 60 * 60 * 1000) {
    return 'next';
  }

  return null;
}

function TimelineEvent({ event, isLast }: { event: TodayEvent; isLast: boolean }) {
  const formatTime = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleTimeString(undefined, {
      hour: 'numeric',
      minute: '2-digit',
    });
  };

  const formatTimeRange = (start: string, end: string) => {
    return `${formatTime(start)} - ${formatTime(end)}`;
  };

  const timingBadge = getTimingBadge(event);

  return (
    <View style={[styles.timelineItem, isLast && styles.timelineItemLast]}>
      <View style={styles.timeColumn}>
        <Text style={styles.timeText}>
          {event.all_day ? 'All day' : formatTime(event.start_time)}
        </Text>
        {!isLast && <View style={styles.timeLine} />}
      </View>
      <View style={styles.eventDetails}>
        <View style={styles.titleRow}>
          <Text style={styles.eventTitle} numberOfLines={1}>
            {event.summary}
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
        {event.location && (
          <Text style={styles.eventLocation} numberOfLines={1}>
            {event.location}
          </Text>
        )}
      </View>
    </View>
  );
}

export function TodayCalendarSection() {
  const { data: events, isLoading, isError } = useTodayEvents();

  if (isLoading) {
    return (
      <View style={styles.container}>
        <Text style={styles.sectionTitle}>TODAY'S SCHEDULE</Text>
        <LoadingSpinner size="small" />
      </View>
    );
  }

  if (isError) {
    return (
      <View style={styles.container}>
        <Text style={styles.sectionTitle}>TODAY'S SCHEDULE</Text>
        <View style={styles.errorState}>
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
        <View style={styles.emptyState}>
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
      <Text style={styles.sectionTitle}>TODAY'S SCHEDULE ({events.length})</Text>
      <View style={styles.timeline}>
        {events.map((event, index) => (
          <TimelineEvent
            key={event.id}
            event={event}
            isLast={index === events.length - 1}
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
    fontWeight: '600',
    color: colors.textSecondary,
    marginBottom: 8,
    marginLeft: 4,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  timeline: {
    backgroundColor: colors.card,
    borderRadius: 12,
    overflow: 'hidden',
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
  timeColumn: {
    width: 70,
    alignItems: 'center',
    paddingRight: 12,
  },
  timeText: {
    fontSize: 13,
    fontWeight: '500',
    color: colors.textSecondary,
  },
  timeLine: {
    flex: 1,
    width: 2,
    backgroundColor: colors.primary + '40',
    marginTop: 8,
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
    marginBottom: 2,
  },
  eventLocation: {
    fontSize: 12,
    color: colors.textSecondary,
    fontStyle: 'italic',
  },
  emptyState: {
    backgroundColor: colors.card,
    borderRadius: 12,
    padding: 24,
    alignItems: 'center',
  },
  emptyText: {
    fontSize: 14,
    fontWeight: '500',
    color: colors.textSecondary,
    marginBottom: 4,
  },
  emptySubtext: {
    fontSize: 12,
    color: colors.textSecondary,
    textAlign: 'center',
  },
  errorState: {
    backgroundColor: colors.card,
    borderRadius: 12,
    padding: 24,
    alignItems: 'center',
  },
  errorText: {
    fontSize: 14,
    fontWeight: '500',
    color: colors.danger,
    marginBottom: 4,
  },
  errorSubtext: {
    fontSize: 12,
    color: colors.textSecondary,
    textAlign: 'center',
  },
});
