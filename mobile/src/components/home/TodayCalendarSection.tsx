import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { LoadingSpinner } from '../common';
import { useTodayEvents } from '../../hooks/useTodayEvents';
import { colors } from '../../theme/colors';
import type { TodayEvent } from '../../types/calendar';

function TimelineEvent({ event }: { event: TodayEvent }) {
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

  return (
    <View style={styles.timelineItem}>
      <View style={styles.timeColumn}>
        <Text style={styles.timeText}>
          {event.all_day ? 'All day' : formatTime(event.start_time)}
        </Text>
        <View style={styles.timeLine} />
      </View>
      <View style={styles.eventDetails}>
        <Text style={styles.eventTitle} numberOfLines={1}>
          {event.summary}
        </Text>
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
        {events.map((event) => (
          <TimelineEvent key={event.id} event={event} />
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
  eventTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 2,
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
