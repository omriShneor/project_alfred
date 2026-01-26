import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { Card } from '../common/Card';
import { colors } from '../../theme/colors';

export function TodoSection() {
  return (
    <View style={styles.container}>
      <Text style={styles.sectionTitle}>TODO LIST</Text>
      <Card style={styles.card}>
        <View style={styles.content}>
          <Text style={styles.comingSoonText}>Coming Soon</Text>
          <Text style={styles.descriptionText}>
            Task management will be available in a future update
          </Text>
        </View>
      </Card>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    marginBottom: 16,
  },
  sectionTitle: {
    fontSize: 12,
    fontWeight: '600',
    color: colors.textSecondary,
    marginBottom: 8,
    marginLeft: 4,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  card: {
    paddingVertical: 24,
    paddingHorizontal: 16,
  },
  content: {
    alignItems: 'center',
  },
  comingSoonText: {
    fontSize: 16,
    fontWeight: '600',
    color: colors.textSecondary,
    marginBottom: 4,
  },
  descriptionText: {
    fontSize: 13,
    color: colors.textSecondary,
    textAlign: 'center',
  },
});
