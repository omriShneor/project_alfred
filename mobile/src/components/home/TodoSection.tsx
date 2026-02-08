import React from 'react';
import { View, Text, StyleSheet, Alert, TextInput, TouchableOpacity } from 'react-native';
import { Card } from '../common/Card';
import { Button } from '../common/Button';
import { Modal } from '../common/Modal';
import { DateTimePicker } from '../common/DateTimePicker';
import { useCreateReminder, useReminders } from '../../hooks/useReminders';
import { colors } from '../../theme/colors';
import type { Reminder, ReminderPriority } from '../../types/reminder';

const priorities: ReminderPriority[] = ['low', 'normal', 'high'];

export function TodoSection() {
  const [visible, setVisible] = React.useState(false);
  const [title, setTitle] = React.useState('');
  const [description, setDescription] = React.useState('');
  const [location, setLocation] = React.useState('');
  const [dueDate, setDueDate] = React.useState<Date | null>(null);
  const [priority, setPriority] = React.useState<ReminderPriority>('normal');

  const createReminder = useCreateReminder();
  const { data: confirmed } = useReminders({ status: 'confirmed' });

  const manualTodos: Reminder[] = React.useMemo(() => {
    const reminders = confirmed || [];
    return reminders.filter((r) => r.source === 'manual');
  }, [confirmed]);

  const resetForm = React.useCallback(() => {
    setTitle('');
    setDescription('');
    setLocation('');
    setDueDate(null);
    setPriority('normal');
  }, []);

  const onCreate = React.useCallback(() => {
    const trimmedTitle = title.trim();
    if (!trimmedTitle) {
      Alert.alert('Missing title', 'Title is required');
      return;
    }

    createReminder.mutate(
      {
        title: trimmedTitle,
        description: description.trim() || undefined,
        location: location.trim() || undefined,
        due_date: dueDate ? dueDate.toISOString() : undefined,
        priority,
      },
      {
        onSuccess: () => {
          setVisible(false);
          resetForm();
        },
        onError: () => {
          Alert.alert('Error', 'Failed to create task');
        },
      }
    );
  }, [createReminder, description, dueDate, location, priority, resetForm, title]);

  const formatDue = (dueDateRaw?: string) => {
    if (!dueDateRaw) {
      return 'No due date';
    }
    return new Date(dueDateRaw).toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  return (
    <View style={styles.container}>
      <View style={styles.headerRow}>
        <Text style={styles.sectionTitle}>TODO LIST</Text>
        <Button
          title="+ Add"
          onPress={() => setVisible(true)}
          size="small"
          style={styles.addButton}
        />
      </View>

      <Card style={styles.card}>
        {manualTodos.length === 0 ? (
          <Text style={styles.emptyText}>No manual tasks yet</Text>
        ) : (
          manualTodos.slice(0, 4).map((todo, index, items) => (
            <View
              key={todo.id}
              style={[
                styles.todoRow,
                index === items.length - 1 && styles.todoRowLast,
              ]}
            >
              <Text style={styles.todoTitle} numberOfLines={1}>{todo.title}</Text>
              <Text style={styles.todoMeta}>{formatDue(todo.due_date)}</Text>
            </View>
          ))
        )}
      </Card>

      <Modal visible={visible} onClose={() => setVisible(false)} title="Add Todo Task">
        <View style={styles.form}>
          <View style={styles.field}>
            <Text style={styles.label}>Title</Text>
            <TextInput
              style={styles.input}
              value={title}
              onChangeText={setTitle}
              placeholder="Task title"
            />
          </View>

          <View style={styles.field}>
            <Text style={styles.label}>Description</Text>
            <TextInput
              style={[styles.input, styles.multiline]}
              value={description}
              onChangeText={setDescription}
              placeholder="Optional description"
              multiline
              numberOfLines={3}
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
              {priorities.map((value) => (
                <TouchableOpacity
                  key={value}
                  onPress={() => setPriority(value)}
                  style={[
                    styles.priorityButton,
                    priority === value && styles.priorityButtonActive,
                  ]}
                >
                  <Text style={styles.priorityText}>{value}</Text>
                </TouchableOpacity>
              ))}
            </View>
          </View>

          <View style={styles.actions}>
            <Button title="Cancel" variant="secondary" onPress={() => setVisible(false)} style={styles.actionButton} />
            <Button
              title="Create"
              variant="primary"
              onPress={onCreate}
              loading={createReminder.isPending}
              style={styles.actionButton}
            />
          </View>
        </View>
      </Modal>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    marginBottom: 16,
  },
  headerRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
    marginLeft: 4,
  },
  sectionTitle: {
    fontSize: 12,
    fontWeight: '600',
    color: colors.textSecondary,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  addButton: {
    minWidth: 72,
  },
  card: {
    paddingVertical: 12,
    paddingHorizontal: 12,
  },
  emptyText: {
    fontSize: 13,
    color: colors.textSecondary,
    textAlign: 'center',
  },
  todoRow: {
    paddingVertical: 8,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: colors.border,
  },
  todoRowLast: {
    borderBottomWidth: 0,
  },
  todoTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.text,
  },
  todoMeta: {
    fontSize: 12,
    color: colors.textSecondary,
    marginTop: 2,
  },
  form: {
    paddingBottom: 8,
  },
  field: {
    marginBottom: 12,
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
  multiline: {
    height: 90,
    textAlignVertical: 'top',
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
  priorityRow: {
    flexDirection: 'row',
    gap: 8,
  },
  priorityButton: {
    flex: 1,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 8,
    paddingVertical: 8,
    alignItems: 'center',
    backgroundColor: colors.background,
  },
  priorityButtonActive: {
    borderColor: colors.primary,
    borderWidth: 2,
  },
  priorityText: {
    color: colors.text,
    fontSize: 12,
    textTransform: 'capitalize',
  },
  actions: {
    flexDirection: 'row',
    gap: 10,
    marginTop: 8,
  },
  actionButton: {
    flex: 1,
  },
});
