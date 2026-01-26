import React, { useState } from 'react';
import { View, Text, TouchableOpacity, StyleSheet, Platform } from 'react-native';
import DateTimePickerModal from 'react-native-modal-datetime-picker';
import { colors } from '../../theme/colors';

interface DateTimePickerProps {
  value: Date | null;
  onChange: (date: Date) => void;
  placeholder?: string;
  label?: string;
  mode?: 'date' | 'time' | 'datetime';
}

export function DateTimePicker({
  value,
  onChange,
  placeholder = 'Select date and time',
  label,
  mode = 'datetime',
}: DateTimePickerProps) {
  const [isVisible, setIsVisible] = useState(false);

  const handleConfirm = (date: Date) => {
    setIsVisible(false);
    onChange(date);
  };

  const handleCancel = () => {
    setIsVisible(false);
  };

  const formatDisplay = (date: Date | null): string => {
    if (!date) return placeholder;

    const options: Intl.DateTimeFormatOptions = {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    };

    if (mode === 'date') {
      delete options.hour;
      delete options.minute;
    } else if (mode === 'time') {
      delete options.year;
      delete options.month;
      delete options.day;
    }

    return date.toLocaleString(undefined, options);
  };

  return (
    <View style={styles.container}>
      {label && <Text style={styles.label}>{label}</Text>}
      <TouchableOpacity
        style={styles.button}
        onPress={() => setIsVisible(true)}
        activeOpacity={0.7}
      >
        <Text style={[styles.buttonText, !value && styles.placeholderText]}>
          {formatDisplay(value)}
        </Text>
      </TouchableOpacity>

      <DateTimePickerModal
        isVisible={isVisible}
        mode={mode}
        date={value || new Date()}
        onConfirm={handleConfirm}
        onCancel={handleCancel}
        display={Platform.OS === 'ios' ? 'spinner' : 'default'}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    marginBottom: 16,
  },
  label: {
    fontSize: 13,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 6,
  },
  button: {
    backgroundColor: colors.background,
    borderRadius: 8,
    paddingHorizontal: 12,
    paddingVertical: 12,
    borderWidth: 1,
    borderColor: colors.border,
  },
  buttonText: {
    fontSize: 14,
    color: colors.text,
  },
  placeholderText: {
    color: colors.textSecondary,
  },
});
