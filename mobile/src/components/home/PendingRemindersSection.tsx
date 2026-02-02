import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { CompactReminderCard } from '../reminders/CompactReminderCard';
import { LoadingSpinner } from '../common';
import { useReminders } from '../../hooks/useReminders';
import { colors } from '../../theme/colors';

export function PendingRemindersSection() {
  const { data: reminders, isLoading } = useReminders({ status: 'pending' });

  if (isLoading) {
    return (
      <View style={styles.container}>
        <Text style={styles.sectionTitle}>PENDING REMINDERS</Text>
        <LoadingSpinner size="small" />
      </View>
    );
  }

  if (!reminders || reminders.length === 0) {
    return (
      <View style={styles.container}>
        <Text style={styles.sectionTitle}>PENDING REMINDERS</Text>
        <View style={styles.emptyState}>
          <Text style={styles.emptyText}>No pending reminders</Text>
          <Text style={styles.emptySubtext}>
            Reminders detected from your messages will appear here
          </Text>
        </View>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <Text style={styles.sectionTitle}>PENDING REMINDERS ({reminders.length})</Text>
      {reminders.map((reminder) => (
        <CompactReminderCard key={reminder.id} reminder={reminder} />
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
