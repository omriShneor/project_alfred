import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { AcceptedReminderCard } from '../reminders/AcceptedReminderCard';
import { LoadingSpinner } from '../common';
import { useReminders } from '../../hooks/useReminders';
import { colors } from '../../theme/colors';
import type { Reminder } from '../../types/reminder';

export function AcceptedRemindersSection() {
  // Fetch both confirmed and synced reminders
  const { data: confirmedReminders, isLoading: loadingConfirmed } = useReminders({ status: 'confirmed' });
  const { data: syncedReminders, isLoading: loadingSynced } = useReminders({ status: 'synced' });

  const isLoading = loadingConfirmed || loadingSynced;

  // Combine and sort by due date
  const reminders: Reminder[] = React.useMemo(() => {
    const all = [...(confirmedReminders || []), ...(syncedReminders || [])];
    return all.sort((a, b) => new Date(a.due_date).getTime() - new Date(b.due_date).getTime());
  }, [confirmedReminders, syncedReminders]);

  if (isLoading) {
    return (
      <View style={styles.container}>
        <Text style={styles.sectionTitle}>ACTIVE REMINDERS</Text>
        <LoadingSpinner size="small" />
      </View>
    );
  }

  if (reminders.length === 0) {
    return null; // Don't show section if no active reminders
  }

  return (
    <View style={styles.container}>
      <Text style={styles.sectionTitle}>ACTIVE REMINDERS ({reminders.length})</Text>
      {reminders.map((reminder) => (
        <AcceptedReminderCard key={reminder.id} reminder={reminder} />
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
});
