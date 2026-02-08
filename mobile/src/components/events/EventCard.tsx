import React, { useState } from 'react';
import { View, Text, TouchableOpacity, StyleSheet, Alert } from 'react-native';
import { Card } from '../common/Card';
import { Badge } from '../common/Badge';
import { Button } from '../common/Button';
import { AttendeeChips } from './AttendeeChips';
import { EditEventModal } from './EditEventModal';
import { MessageContextModal } from './MessageContextModal';
import { useConfirmEvent, useRejectEvent } from '../../hooks/useEvents';
import { colors } from '../../theme/colors';
import type { CalendarEvent } from '../../types/event';

interface EventCardProps {
  event: CalendarEvent;
}

export function EventCard({ event }: EventCardProps) {
  const [showReasoning, setShowReasoning] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [showContextModal, setShowContextModal] = useState(false);
  const eventTitle = event.title?.trim() || 'Untitled event';
  const eventDescription = event.description?.trim() || '';

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

  const isPending = event.status === 'pending';
  const isLoading = confirmEvent.isPending || rejectEvent.isPending;

  return (
    <>
      <Card>
        <View style={styles.header}>
          <View style={styles.badges}>
            <Badge label={event.action_type} variant={event.action_type} />
            <Badge label={event.status} variant="status" status={event.status} />
          </View>
          {event.channel_name && (
            <Text style={styles.channelName} numberOfLines={1}>
              {event.channel_name}
            </Text>
          )}
        </View>

        <Text style={styles.title}>{eventTitle}</Text>

        {eventDescription ? (
          <View style={styles.details}>
            <View style={styles.detailRow}>
              <Text style={styles.detailLabel}>Details:</Text>
              <Text style={styles.detailValue}>{eventDescription}</Text>
            </View>
          </View>
        ) : null}

        <View style={styles.details}>
          <View style={styles.detailRow}>
            <Text style={styles.detailLabel}>When:</Text>
            <Text style={styles.detailValue}>
              {formatDateTime(event.start_time)}
              {event.end_time && ` - ${formatDateTime(event.end_time)}`}
            </Text>
          </View>

          {event.location && (
            <View style={styles.detailRow}>
              <Text style={styles.detailLabel}>Where:</Text>
              <Text style={styles.detailValue}>{event.location}</Text>
            </View>
          )}
        </View>

        {event.attendees && event.attendees.length > 0 && (
          <AttendeeChips attendees={event.attendees} />
        )}

        {event.llm_reasoning && (
          <TouchableOpacity
            style={styles.reasoningToggle}
            onPress={() => setShowReasoning(!showReasoning)}
          >
            <Text style={styles.reasoningLabel}>
              AI Reasoning {showReasoning ? '▲' : '▼'}
            </Text>
          </TouchableOpacity>
        )}

        {showReasoning && event.llm_reasoning && (
          <View style={styles.reasoning}>
            <Text style={styles.reasoningText}>{event.llm_reasoning}</Text>
          </View>
        )}

        <View style={styles.actions}>
          <Button
            title="View Context"
            onPress={() => setShowContextModal(true)}
            variant="outline"
            size="small"
            style={styles.actionButton}
          />

          {isPending && (
            <>
              <Button
                title="Edit"
                onPress={() => setShowEditModal(true)}
                variant="secondary"
                size="small"
                style={styles.actionButton}
              />
              <Button
                title="Confirm"
                onPress={handleConfirm}
                variant="success"
                size="small"
                loading={confirmEvent.isPending}
                disabled={isLoading}
                style={styles.actionButton}
              />
              <Button
                title="Reject"
                onPress={handleReject}
                variant="danger"
                size="small"
                loading={rejectEvent.isPending}
                disabled={isLoading}
                style={styles.actionButton}
              />
            </>
          )}
        </View>
      </Card>

      <EditEventModal
        visible={showEditModal}
        onClose={() => setShowEditModal(false)}
        event={event}
      />

      <MessageContextModal
        visible={showContextModal}
        onClose={() => setShowContextModal(false)}
        channelId={event.channel_id}
        channelName={event.channel_name}
      />
    </>
  );
}

const styles = StyleSheet.create({
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  badges: {
    flexDirection: 'row',
    gap: 8,
  },
  channelName: {
    fontSize: 12,
    color: colors.textSecondary,
    maxWidth: 120,
  },
  title: {
    fontSize: 16,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 8,
  },
  details: {
    marginBottom: 8,
  },
  detailRow: {
    flexDirection: 'row',
    marginBottom: 4,
  },
  detailLabel: {
    fontSize: 13,
    color: colors.textSecondary,
    width: 50,
  },
  detailValue: {
    fontSize: 13,
    color: colors.text,
    flex: 1,
  },
  reasoningToggle: {
    paddingVertical: 8,
  },
  reasoningLabel: {
    fontSize: 12,
    color: colors.primary,
    fontWeight: '500',
  },
  reasoning: {
    backgroundColor: colors.background,
    borderRadius: 6,
    padding: 10,
    marginBottom: 8,
  },
  reasoningText: {
    fontSize: 12,
    color: colors.textSecondary,
    lineHeight: 18,
  },
  actions: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 8,
    marginTop: 12,
    paddingTop: 12,
    borderTopWidth: 1,
    borderTopColor: colors.border,
  },
  actionButton: {
    minWidth: 70,
  },
});
