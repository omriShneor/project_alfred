import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { CompactEventCard } from '../events/CompactEventCard';
import { LoadingSpinner } from '../common';
import { useEvents } from '../../hooks/useEvents';
import { colors } from '../../theme/colors';

export function PendingEventsSection() {
  const { data: events, isLoading } = useEvents({ status: 'pending' });
  const sortedEvents = React.useMemo(() => {
    if (!events) {
      return [];
    }

    return [...events].sort(
      (a, b) =>
        new Date(a.start_time).getTime() - new Date(b.start_time).getTime()
    );
  }, [events]);

  if (isLoading) {
    return (
      <View style={styles.container}>
        <Text style={styles.sectionTitle}>PENDING EVENTS</Text>
        <LoadingSpinner size="small" />
      </View>
    );
  }

  if (sortedEvents.length === 0) {
    return (
      <View style={styles.container}>
        <Text style={styles.sectionTitle}>PENDING EVENTS</Text>
        <View style={styles.emptyState}>
          <Text style={styles.emptyText}>No pending events</Text>
          <Text style={styles.emptySubtext}>
            Events detected from your tracked contacts/groups will appear here
          </Text>
        </View>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <Text style={styles.sectionTitle}>PENDING EVENTS ({sortedEvents.length})</Text>
      {sortedEvents.map((event) => (
        <CompactEventCard key={event.id} event={event} />
      ))}
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
});
