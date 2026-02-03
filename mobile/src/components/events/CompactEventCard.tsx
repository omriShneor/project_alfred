import React, { useState } from 'react';
import { View, Text, StyleSheet, Alert } from 'react-native';
import { Card } from '../common/Card';
import { IconButton } from '../common/IconButton';
import { EditEventModal } from './EditEventModal';
import { useConfirmEvent, useRejectEvent } from '../../hooks/useEvents';
import { colors } from '../../theme/colors';
import type { CalendarEvent } from '../../types/event';

interface CompactEventCardProps {
  event: CalendarEvent;
}

export function CompactEventCard({ event }: CompactEventCardProps) {
  const [showEditModal, setShowEditModal] = useState(false);

  const confirmEvent = useConfirmEvent();
  const rejectEvent = useRejectEvent();

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

  const handleConfirm = () => {
    Alert.alert(
      'Confirm Event',
      'Sync this event to Google Calendar?',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Confirm',
          onPress: () => {
            confirmEvent.mutate(event.id, {
              onError: () => {
                Alert.alert('Error', 'Failed to sync event');
              },
            });
          },
        },
      ]
    );
  };

  const handleReject = () => {
    Alert.alert(
      'Reject Event',
      'Are you sure you want to reject this event?',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Reject',
          style: 'destructive',
          onPress: () => {
            rejectEvent.mutate(event.id, {
              onError: () => {
                Alert.alert('Error', 'Failed to reject event');
              },
            });
          },
        },
      ]
    );
  };

  const isLoading = confirmEvent.isPending || rejectEvent.isPending;

  return (
    <>
      <Card style={styles.card}>
        <View style={styles.content}>
          <View style={styles.info}>
            <View style={styles.titleRow}>
              <Text style={styles.title} numberOfLines={1}>
                {event.title}
              </Text>
              {event.channel_name && (
                <Text style={styles.channel}>#{event.channel_name}</Text>
              )}
            </View>
            <Text style={styles.dateTime}>{formatDateTime(event.start_time)}</Text>
          </View>

          <View style={styles.actions}>
            <IconButton
              testID="edit-button"
              icon="edit-2"
              onPress={() => setShowEditModal(true)}
              color={colors.primary}
              backgroundColor={colors.primary + '15'}
              size={16}
              disabled={isLoading}
            />
            <IconButton
              testID="reject-button"
              icon="x"
              onPress={handleReject}
              color={colors.danger}
              backgroundColor={colors.danger + '15'}
              size={16}
              disabled={isLoading}
              loading={rejectEvent.isPending}
            />
            <IconButton
              testID="confirm-button"
              icon="check"
              onPress={handleConfirm}
              color={colors.success}
              backgroundColor={colors.success + '15'}
              size={16}
              disabled={isLoading}
              loading={confirmEvent.isPending}
            />
          </View>
        </View>
      </Card>

      <EditEventModal
        visible={showEditModal}
        onClose={() => setShowEditModal(false)}
        event={event}
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
  },
  actions: {
    flexDirection: 'row',
    gap: 8,
  },
});
