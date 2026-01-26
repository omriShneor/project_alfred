import React from 'react';
import { ScrollView, StyleSheet, RefreshControl } from 'react-native';
import { useQueryClient } from '@tanstack/react-query';
import { TodoSection, PendingEventsSection, TodayCalendarSection } from '../components/home';
import { colors } from '../theme/colors';

export function HomeScreen() {
  const queryClient = useQueryClient();
  const [refreshing, setRefreshing] = React.useState(false);

  const onRefresh = React.useCallback(async () => {
    setRefreshing(true);
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['events'] }),
      queryClient.invalidateQueries({ queryKey: ['todayEvents'] }),
    ]);
    setRefreshing(false);
  }, [queryClient]);

  return (
    <ScrollView
      style={styles.container}
      contentContainerStyle={styles.content}
      refreshControl={
        <RefreshControl
          refreshing={refreshing}
          onRefresh={onRefresh}
          colors={[colors.primary]}
          tintColor={colors.primary}
        />
      }
    >
      <TodoSection />
      <PendingEventsSection />
      <TodayCalendarSection />
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.background,
  },
  content: {
    padding: 16,
  },
});
