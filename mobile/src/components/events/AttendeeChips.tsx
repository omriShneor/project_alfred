import React from 'react';
import { View, Text, StyleSheet, ScrollView } from 'react-native';
import { colors } from '../../theme/colors';
import type { Attendee } from '../../types/event';

interface AttendeeChipsProps {
  attendees: Attendee[];
}

export function AttendeeChips({ attendees }: AttendeeChipsProps) {
  if (attendees.length === 0) {
    return null;
  }

  return (
    <View style={styles.container}>
      <Text style={styles.label}>Attendees:</Text>
      <ScrollView horizontal showsHorizontalScrollIndicator={false}>
        <View style={styles.chips}>
          {attendees.map((attendee) => (
            <View key={attendee.id} style={styles.chip}>
              <Text style={styles.chipText}>{attendee.name}</Text>
            </View>
          ))}
        </View>
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    marginTop: 8,
  },
  label: {
    fontSize: 12,
    color: colors.textSecondary,
    marginBottom: 6,
  },
  chips: {
    flexDirection: 'row',
    gap: 8,
  },
  chip: {
    backgroundColor: colors.primary + '15',
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 12,
  },
  chipText: {
    fontSize: 12,
    color: colors.primary,
    fontWeight: '500',
  },
});
