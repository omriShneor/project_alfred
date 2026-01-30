import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Select } from '../common/Select';
import { useCalendars } from '../../hooks/useEvents';

interface CalendarPickerProps {
  value: string;
  onChange: (value: string) => void;
  enabled?: boolean;
}

export function CalendarPicker({ value, onChange, enabled = true }: CalendarPickerProps) {
  const { data: calendars, isLoading } = useCalendars(enabled);

  const options = [
    { label: 'Select calendar...', value: '' },
    ...(calendars?.map((cal) => ({
      label: cal.primary ? `${cal.summary} (Primary)` : cal.summary,
      value: cal.id,
    })) || []),
  ];

  if (isLoading) {
    return (
      <View style={styles.container}>
        <Select
          options={[{ label: 'Loading calendars...', value: '' }]}
          value=""
          onChange={() => {}}
          placeholder="Loading..."
        />
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <Select
        options={options}
        value={value}
        onChange={onChange}
        placeholder="Select calendar"
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
});
