import React, { useState } from 'react';
import { View, Text, StyleSheet, Alert } from 'react-native';
import { Card } from '../common/Card';
import { IconButton } from '../common/IconButton';
import { EditReminderModal } from './EditReminderModal';
import { useConfirmReminder, useRejectReminder } from '../../hooks/useReminders';
import { colors } from '../../theme/colors';
import type { Reminder } from '../../types/reminder';

interface CompactReminderCardProps {
  reminder: Reminder;
}

const priorityColors: Record<string, string> = {
  low: colors.textSecondary,
  normal: colors.primary,
  high: colors.danger,
};

export function CompactReminderCard({ reminder }: CompactReminderCardProps) {
  const [showEditModal, setShowEditModal] = useState(false);

  const confirmReminder = useConfirmReminder();
  const rejectReminder = useRejectReminder();

  const formatDateTime = (dateString?: string) => {
    if (!dateString) {
      return 'No due date';
    }
    const date = new Date(dateString);
    return date.toLocaleString(undefined, {
      weekday: 'short',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const handleConfirm = () => {
    Alert.alert(
      'Confirm Reminder',
      'Accept this reminder?',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Confirm',
          onPress: () => {
            confirmReminder.mutate(reminder.id, {
              onError: () => {
                Alert.alert('Error', 'Failed to confirm reminder');
              },
            });
          },
        },
      ]
    );
  };

  const handleReject = () => {
    Alert.alert(
      'Reject Reminder',
      'Are you sure you want to reject this reminder?',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Reject',
          style: 'destructive',
          onPress: () => {
            rejectReminder.mutate(reminder.id, {
              onError: () => {
                Alert.alert('Error', 'Failed to reject reminder');
              },
            });
          },
        },
      ]
    );
  };

  const isLoading = confirmReminder.isPending || rejectReminder.isPending;

  return (
    <>
      <Card style={styles.card}>
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
              {reminder.channel_name && (
                <Text style={styles.channel}>#{reminder.channel_name}</Text>
              )}
            </View>
            <Text style={styles.dateTime}>Due: {formatDateTime(reminder.due_date)}</Text>
            {reminder.location ? (
              <Text style={styles.location}>Location: {reminder.location}</Text>
            ) : null}
          </View>

          <View style={styles.actions}>
            <IconButton
              icon="edit-2"
              onPress={() => setShowEditModal(true)}
              color={colors.primary}
              backgroundColor={colors.primary + '15'}
              size={16}
              disabled={isLoading}
            />
            <IconButton
              icon="x"
              onPress={handleReject}
              color={colors.danger}
              backgroundColor={colors.danger + '15'}
              size={16}
              disabled={isLoading}
              loading={rejectReminder.isPending}
            />
            <IconButton
              icon="check"
              onPress={handleConfirm}
              color={colors.success}
              backgroundColor={colors.success + '15'}
              size={16}
              disabled={isLoading}
              loading={confirmReminder.isPending}
            />
          </View>
        </View>
      </Card>

      <EditReminderModal
        visible={showEditModal}
        onClose={() => setShowEditModal(false)}
        reminder={reminder}
      />
    </>
  );
}

const styles = StyleSheet.create({
  card: {
    marginBottom: 8,
    paddingVertical: 12,
    paddingHorizontal: 12,
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
  channel: {
    fontSize: 12,
    color: colors.textSecondary,
    marginLeft: 8,
  },
  dateTime: {
    fontSize: 13,
    color: colors.textSecondary,
    marginLeft: 16,
  },
  location: {
    fontSize: 12,
    color: colors.textSecondary,
    marginLeft: 16,
    marginTop: 2,
  },
  actions: {
    flexDirection: 'row',
    gap: 8,
  },
});
