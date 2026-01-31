import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { colors } from '../../theme/colors';
import type { DiscoverableChannel } from '../../types/channel';

interface ChannelStatsProps {
  channels: DiscoverableChannel[];
}

export function ChannelStats({ channels }: ChannelStatsProps) {
  const total = channels.length;
  const tracked = channels.filter((c) => c.is_tracked).length;

  const stats = [
    { label: 'Total Contacts', value: total },
    { label: 'Tracked', value: tracked },
  ];

  return (
    <View style={styles.container}>
      {stats.map((stat, index) => (
        <View key={stat.label} style={styles.stat}>
          <Text style={styles.value}>{stat.value}</Text>
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
    color: colors.text,
  },
  label: {
    fontSize: 11,
    color: colors.textSecondary,
    marginTop: 2,
  },
});
