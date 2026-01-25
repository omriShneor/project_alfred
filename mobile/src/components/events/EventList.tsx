import React from 'react';
import { FlatList, Text, View, StyleSheet, RefreshControl } from 'react-native';
import { EventCard } from './EventCard';
import { colors } from '../../theme/colors';
import type { CalendarEvent } from '../../types/event';

interface EventListProps {
  events: CalendarEvent[];
  refreshing?: boolean;
  onRefresh?: () => void;
}

export function EventList({ events, refreshing, onRefresh }: EventListProps) {
  if (events.length === 0) {
    return (
      <View style={styles.empty}>
        <Text style={styles.emptyText}>No events found</Text>
        <Text style={styles.emptySubtext}>
          Events detected from your tracked channels will appear here
        </Text>
      </View>
    );
  }

  return (
    <FlatList
      data={events}
      keyExtractor={(item) => item.id.toString()}
      renderItem={({ item }) => <EventCard event={item} />}
      contentContainerStyle={styles.list}
      refreshControl={
        onRefresh ? (
          <RefreshControl refreshing={refreshing || false} onRefresh={onRefresh} />
        ) : undefined
      }
    />
  );
}

const styles = StyleSheet.create({
  list: {
    paddingBottom: 20,
  },
  empty: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 40,
  },
  emptyText: {
    fontSize: 16,
    fontWeight: '600',
    color: colors.text,
  },
  emptySubtext: {
    fontSize: 14,
    color: colors.textSecondary,
    marginTop: 8,
    textAlign: 'center',
  },
});
