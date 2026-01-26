import React, { useState, useEffect } from 'react';
import { View, Text, TextInput, StyleSheet, Alert } from 'react-native';
import { Modal } from '../common/Modal';
import { Button } from '../common/Button';
import { DateTimePicker } from '../common/DateTimePicker';
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
  const [startTime, setStartTime] = useState<Date | null>(null);
  const [endTime, setEndTime] = useState<Date | null>(null);

  const updateEvent = useUpdateEvent();

  useEffect(() => {
    if (event) {
      setTitle(event.title);
      setLocation(event.location || '');
      setDescription(event.description || '');
      setStartTime(new Date(event.start_time));
      setEndTime(event.end_time ? new Date(event.end_time) : null);
    }
  }, [event]);

  const handleSave = () => {
    if (!event) return;

    if (!startTime) {
      Alert.alert('Invalid Date', 'Please select a start time');
      return;
    }

    updateEvent.mutate(
      {
        id: event.id,
        data: {
          title,
          location: location || undefined,
          description: description || undefined,
          start_time: startTime.toISOString(),
          end_time: endTime ? endTime.toISOString() : undefined,
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

        <DateTimePicker
          label="Start Time"
          value={startTime}
          onChange={setStartTime}
          placeholder="Select start time"
        />

        <DateTimePicker
          label="End Time"
          value={endTime}
          onChange={setEndTime}
          placeholder="Select end time (optional)"
        />

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
