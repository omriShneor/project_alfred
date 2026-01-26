import React, { useState, useEffect } from 'react';
import { View, Text, TextInput, StyleSheet, Alert } from 'react-native';
import { Modal } from '../common/Modal';
import { Button } from '../common/Button';
import { useUpdateEvent } from '../../hooks/useEvents';
import { colors } from '../../theme/colors';
import type { CalendarEvent } from '../../types/event';

interface EditEventModalProps {
  visible: boolean;
  onClose: () => void;
  event: CalendarEvent | null;
}

export function EditEventModal({ visible, onClose, event }: EditEventModalProps) {
  const [title, setTitle] = useState('');
  const [location, setLocation] = useState('');
  const [description, setDescription] = useState('');
  const [startTime, setStartTime] = useState('');
  const [endTime, setEndTime] = useState('');

  const updateEvent = useUpdateEvent();

  useEffect(() => {
    if (event) {
      setTitle(event.title);
      setLocation(event.location || '');
      setDescription(event.description || '');
      setStartTime(formatDateForInput(event.start_time));
      setEndTime(event.end_time ? formatDateForInput(event.end_time) : '');
    }
  }, [event]);

  const formatDateForInput = (dateString: string) => {
    const date = new Date(dateString);
    // Format: YYYY-MM-DD HH:MM
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    return `${year}-${month}-${day} ${hours}:${minutes}`;
  };

  const parseInputDate = (input: string): string | null => {
    // Parse YYYY-MM-DD HH:MM format
    const match = input.match(/^(\d{4})-(\d{2})-(\d{2})\s+(\d{2}):(\d{2})$/);
    if (!match) return null;

    const [, year, month, day, hours, minutes] = match;
    const date = new Date(
      parseInt(year),
      parseInt(month) - 1,
      parseInt(day),
      parseInt(hours),
      parseInt(minutes)
    );
    return date.toISOString();
  };

  const handleSave = () => {
    if (!event) return;

    const parsedStart = parseInputDate(startTime);
    if (!parsedStart) {
      Alert.alert('Invalid Date', 'Please enter start time as YYYY-MM-DD HH:MM');
      return;
    }

    const parsedEnd = endTime ? parseInputDate(endTime) : undefined;
    if (endTime && !parsedEnd) {
      Alert.alert('Invalid Date', 'Please enter end time as YYYY-MM-DD HH:MM');
      return;
    }

    updateEvent.mutate(
      {
        id: event.id,
        data: {
          title,
          location: location || undefined,
          description: description || undefined,
          start_time: parsedStart,
          end_time: parsedEnd ?? undefined,
        },
      },
      {
        onSuccess: () => {
          onClose();
        },
        onError: (error) => {
          Alert.alert('Error', 'Failed to update event');
        },
      }
    );
  };

  return (
    <Modal visible={visible} onClose={onClose} title="Edit Event">
      <View style={styles.form}>
        <View style={styles.field}>
          <Text style={styles.label}>Title</Text>
          <TextInput
            style={styles.input}
            value={title}
            onChangeText={setTitle}
            placeholder="Event title"
          />
        </View>

        <View style={styles.field}>
          <Text style={styles.label}>Start Time (YYYY-MM-DD HH:MM)</Text>
          <TextInput
            style={styles.input}
            value={startTime}
            onChangeText={setStartTime}
            placeholder="2024-01-15 14:00"
          />
        </View>

        <View style={styles.field}>
          <Text style={styles.label}>End Time (YYYY-MM-DD HH:MM)</Text>
          <TextInput
            style={styles.input}
            value={endTime}
            onChangeText={setEndTime}
            placeholder="2024-01-15 15:00"
          />
        </View>

        <View style={styles.field}>
          <Text style={styles.label}>Location</Text>
          <TextInput
            style={styles.input}
            value={location}
            onChangeText={setLocation}
            placeholder="Event location"
          />
        </View>

        <View style={styles.field}>
          <Text style={styles.label}>Description</Text>
          <TextInput
            style={[styles.input, styles.textArea]}
            value={description}
            onChangeText={setDescription}
            placeholder="Event description"
            multiline
            numberOfLines={4}
          />
        </View>

        <View style={styles.buttons}>
          <Button
            title="Cancel"
            onPress={onClose}
            variant="secondary"
            style={styles.button}
          />
          <Button
            title="Save"
            onPress={handleSave}
            variant="primary"
            loading={updateEvent.isPending}
            style={styles.button}
          />
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  form: {
    paddingBottom: 20,
  },
  field: {
    marginBottom: 16,
  },
  label: {
    fontSize: 13,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 6,
  },
  input: {
    backgroundColor: colors.background,
    borderRadius: 8,
    paddingHorizontal: 12,
    paddingVertical: 10,
    fontSize: 14,
    color: colors.text,
    borderWidth: 1,
    borderColor: colors.border,
  },
  textArea: {
    height: 100,
    textAlignVertical: 'top',
  },
  buttons: {
    flexDirection: 'row',
    gap: 12,
    marginTop: 8,
  },
  button: {
    flex: 1,
  },
});
