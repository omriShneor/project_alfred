import React, { useState, useEffect } from 'react';
import { View, Text, TextInput, StyleSheet, Alert, TouchableOpacity } from 'react-native';
import { Modal } from '../common/Modal';
import { Button } from '../common/Button';
import { DateTimePicker } from '../common/DateTimePicker';
import { useUpdateReminder } from '../../hooks/useReminders';
import { colors } from '../../theme/colors';
import type { Reminder, ReminderPriority } from '../../types/reminder';

interface EditReminderModalProps {
  visible: boolean;
  onClose: () => void;
  reminder: Reminder | null;
}

const priorities: { value: ReminderPriority; label: string; color: string }[] = [
  { value: 'low', label: 'Low', color: colors.textSecondary },
  { value: 'normal', label: 'Normal', color: colors.primary },
  { value: 'high', label: 'High', color: colors.danger },
];

export function EditReminderModal({ visible, onClose, reminder }: EditReminderModalProps) {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [location, setLocation] = useState('');
  const [dueDate, setDueDate] = useState<Date | null>(null);
  const [priority, setPriority] = useState<ReminderPriority>('normal');

  const updateReminder = useUpdateReminder();

  useEffect(() => {
    if (reminder) {
      setTitle(reminder.title);
      setDescription(reminder.description || '');
      setLocation(reminder.location || '');
      setDueDate(reminder.due_date ? new Date(reminder.due_date) : null);
      setPriority(reminder.priority);
    }
  }, [reminder]);

  const handleSave = () => {
    if (!reminder) return;

    updateReminder.mutate(
      {
        id: reminder.id,
        data: {
          title,
          description: description || undefined,
          location: location || '',
          due_date: dueDate ? dueDate.toISOString() : '',
          priority,
        },
      },
      {
        onSuccess: () => {
          onClose();
        },
        onError: () => {
          Alert.alert('Error', 'Failed to update reminder');
        },
      }
    );
  };

  return (
    <Modal visible={visible} onClose={onClose} title="Edit Reminder">
      <View style={styles.form}>
        <View style={styles.field}>
          <Text style={styles.label}>Title</Text>
          <TextInput
            style={styles.input}
            value={title}
            onChangeText={setTitle}
            placeholder="Reminder title"
          />
        </View>

        <DateTimePicker
          label="Due Date"
          value={dueDate}
          onChange={setDueDate}
          placeholder="Optional"
        />
        <TouchableOpacity onPress={() => setDueDate(null)} style={styles.clearDateButton}>
          <Text style={styles.clearDateText}>Clear due date</Text>
        </TouchableOpacity>

        <View style={styles.field}>
          <Text style={styles.label}>Priority</Text>
          <View style={styles.priorityRow}>
            {priorities.map((p) => (
              <TouchableOpacity
                key={p.value}
                style={[
                  styles.priorityButton,
                  priority === p.value && styles.priorityButtonActive,
                  priority === p.value && { borderColor: p.color },
                ]}
                onPress={() => setPriority(p.value)}
              >
                <View
                  style={[styles.priorityDot, { backgroundColor: p.color }]}
                />
                <Text
                  style={[
                    styles.priorityLabel,
                    priority === p.value && { color: p.color },
                  ]}
                >
                  {p.label}
                </Text>
              </TouchableOpacity>
            ))}
          </View>
        </View>

        <View style={styles.field}>
          <Text style={styles.label}>Description</Text>
          <TextInput
            style={[styles.input, styles.textArea]}
            value={description}
            onChangeText={setDescription}
            placeholder="Reminder description"
            multiline
            numberOfLines={4}
          />
        </View>

        <View style={styles.field}>
          <Text style={styles.label}>Location</Text>
          <TextInput
            style={styles.input}
            value={location}
            onChangeText={setLocation}
            placeholder="Optional location"
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
            loading={updateReminder.isPending}
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
  priorityRow: {
    flexDirection: 'row',
    gap: 8,
  },
  clearDateButton: {
    marginTop: -8,
    marginBottom: 8,
  },
  clearDateText: {
    color: colors.primary,
    fontSize: 12,
    fontWeight: '500',
  },
  priorityButton: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 10,
    paddingHorizontal: 12,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.background,
  },
  priorityButtonActive: {
    borderWidth: 2,
  },
  priorityDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    marginRight: 6,
  },
  priorityLabel: {
    fontSize: 13,
    color: colors.text,
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
