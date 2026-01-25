import React from 'react';
import { View, Text, StyleSheet, FlatList } from 'react-native';
import { Modal } from '../common/Modal';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { useChannelHistory } from '../../hooks/useEvents';
import { colors } from '../../theme/colors';

interface MessageContextModalProps {
  visible: boolean;
  onClose: () => void;
  channelId: number;
  channelName?: string;
}

export function MessageContextModal({
  visible,
  onClose,
  channelId,
  channelName,
}: MessageContextModalProps) {
  const { data: messages, isLoading } = useChannelHistory(visible ? channelId : 0);

  const formatTime = (timestamp: string) => {
    const date = new Date(timestamp);
    return date.toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  return (
    <Modal
      visible={visible}
      onClose={onClose}
      title={`Messages${channelName ? ` - ${channelName}` : ''}`}
    >
      {isLoading ? (
        <LoadingSpinner message="Loading messages..." />
      ) : messages && messages.length > 0 ? (
        <FlatList
          data={messages}
          keyExtractor={(item) => item.id.toString()}
          renderItem={({ item }) => (
            <View style={styles.message}>
              <View style={styles.messageHeader}>
                <Text style={styles.sender}>{item.sender_name}</Text>
                <Text style={styles.time}>{formatTime(item.timestamp)}</Text>
              </View>
              <Text style={styles.text}>{item.message_text}</Text>
            </View>
          )}
          contentContainerStyle={styles.list}
          scrollEnabled={false}
        />
      ) : (
        <View style={styles.empty}>
          <Text style={styles.emptyText}>No messages found</Text>
        </View>
      )}
    </Modal>
  );
}

const styles = StyleSheet.create({
  list: {
    paddingBottom: 20,
  },
  message: {
    backgroundColor: colors.background,
    borderRadius: 8,
    padding: 12,
    marginBottom: 8,
  },
  messageHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginBottom: 6,
  },
  sender: {
    fontSize: 13,
    fontWeight: '600',
    color: colors.primary,
  },
  time: {
    fontSize: 11,
    color: colors.textSecondary,
  },
  text: {
    fontSize: 14,
    color: colors.text,
    lineHeight: 20,
  },
  empty: {
    padding: 40,
    alignItems: 'center',
  },
  emptyText: {
    fontSize: 14,
    color: colors.textSecondary,
  },
});
