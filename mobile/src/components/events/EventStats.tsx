import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { colors, statusColors } from '../../theme/colors';
import type { CalendarEvent } from '../../types/event';

interface EventStatsProps {
  events: CalendarEvent[];
}

export function EventStats({ events }: EventStatsProps) {
  const pending = events.filter((e) => e.status === 'pending').length;
  const synced = events.filter((e) => e.status === 'synced').length;
  const total = events.length;

  const stats = [
    { label: 'Pending', value: pending, color: statusColors.pending },
    { label: 'Synced', value: synced, color: statusColors.synced },
    { label: 'Total', value: total, color: colors.text },
  ];

  return (
    <View style={styles.container}>
      {stats.map((stat) => (
        <View key={stat.label} style={styles.stat}>
          <Text style={[styles.value, { color: stat.color }]}>{stat.value}</Text>
          <Text style={styles.label}>{stat.label}</Text>
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    backgroundColor: colors.card,
    borderRadius: 8,
    padding: 12,
    marginBottom: 12,
  },
  stat: {
    flex: 1,
    alignItems: 'center',
  },
  value: {
    fontSize: 18,
    fontWeight: '700',
  },
  label: {
    fontSize: 11,
    color: colors.textSecondary,
    marginTop: 2,
  },
});
