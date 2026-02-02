import React from 'react';
import { View, Text, StyleSheet, Alert } from 'react-native';
import { Card } from '../common/Card';
import { IconButton } from '../common/IconButton';
import { useCompleteReminder, useDismissReminder } from '../../hooks/useReminders';
import { colors } from '../../theme/colors';
import type { Reminder } from '../../types/reminder';

interface AcceptedReminderCardProps {
  reminder: Reminder;
}

const priorityColors: Record<string, string> = {
  low: colors.textSecondary,
  normal: colors.primary,
  high: colors.danger,
};

export function AcceptedReminderCard({ reminder }: AcceptedReminderCardProps) {
  const completeReminder = useCompleteReminder();
  const dismissReminder = useDismissReminder();

  const formatDateTime = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleString(undefined, {
      weekday: 'short',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const isOverdue = () => {
    const now = new Date();
    const dueDate = new Date(reminder.due_date);
    return dueDate < now;
  };

  const handleComplete = () => {
    completeReminder.mutate(reminder.id, {
      onError: () => {
        Alert.alert('Error', 'Failed to complete reminder');
      },
    });
  };

  const handleDismiss = () => {
    Alert.alert(
      'Dismiss Reminder',
      'Are you sure you want to dismiss this reminder?',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Dismiss',
          style: 'destructive',
          onPress: () => {
            dismissReminder.mutate(reminder.id, {
              onError: () => {
                Alert.alert('Error', 'Failed to dismiss reminder');
              },
            });
          },
        },
      ]
    );
  };

  const isLoading = completeReminder.isPending || dismissReminder.isPending;

  return (
    <Card style={[styles.card, isOverdue() && styles.overdueCard]}>
      <View style={styles.content}>
        <View style={styles.info}>
          <View style={styles.titleRow}>
            <View
              style={[
                styles.priorityIndicator,
                { backgroundColor: priorityColors[reminder.priority] || colors.primary },
              ]}
            />
            <Text style={styles.title} numberOfLines={1}>
              {reminder.title}
            </Text>
          </View>
          <Text style={[styles.dateTime, isOverdue() && styles.overdueText]}>
            {isOverdue() ? 'Overdue: ' : 'Due: '}{formatDateTime(reminder.due_date)}
          </Text>
        </View>

        <View style={styles.actions}>
          <IconButton
            icon="x"
            onPress={handleDismiss}
            color={colors.textSecondary}
            backgroundColor={colors.textSecondary + '15'}
            size={16}
            disabled={isLoading}
            loading={dismissReminder.isPending}
          />
          <IconButton
            icon="check-circle"
            onPress={handleComplete}
            color={colors.success}
            backgroundColor={colors.success + '15'}
            size={16}
            disabled={isLoading}
            loading={completeReminder.isPending}
          />
        </View>
      </View>
    </Card>
  );
}

const styles = StyleSheet.create({
  card: {
    marginBottom: 8,
    paddingVertical: 12,
    paddingHorizontal: 12,
  },
  overdueCard: {
    borderLeftWidth: 3,
    borderLeftColor: colors.danger,
  },
  content: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  info: {
    flex: 1,
    marginRight: 12,
  },
  titleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 4,
  },
  priorityIndicator: {
    width: 8,
    height: 8,
    borderRadius: 4,
    marginRight: 8,
  },
  title: {
    fontSize: 15,
    fontWeight: '600',
    color: colors.text,
    flex: 1,
  },
  dateTime: {
    fontSize: 13,
    color: colors.textSecondary,
    marginLeft: 16,
  },
  overdueText: {
    color: colors.danger,
  },
  actions: {
    flexDirection: 'row',
    gap: 8,
  },
});
