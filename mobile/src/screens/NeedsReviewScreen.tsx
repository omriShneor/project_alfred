import React from 'react';
import {
  View,
  Text,
  ScrollView,
  StyleSheet,
  RefreshControl,
  TouchableOpacity,
  LayoutChangeEvent,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useNavigation, type NavigationProp, type ParamListBase } from '@react-navigation/native';
import { useQueryClient } from '@tanstack/react-query';
import { Ionicons } from '@expo/vector-icons';
import { Button, Card, LoadingSpinner } from '../components/common';
import {
  PendingRemindersSection,
  PendingEventsSection,
} from '../components/home';
import { useEvents } from '../hooks/useEvents';
import { useReminders } from '../hooks/useReminders';
import { colors } from '../theme/colors';

export function NeedsReviewScreen() {
  const navigation = useNavigation<NavigationProp<ParamListBase>>();
  const queryClient = useQueryClient();
  const scrollViewRef = React.useRef<ScrollView>(null);

  const [remindersY, setRemindersY] = React.useState(0);
  const [eventsY, setEventsY] = React.useState(0);
  const [refreshing, setRefreshing] = React.useState(false);

  const { data: pendingReminders, isLoading: loadingReminders } =
    useReminders({ status: 'pending' });
  const { data: pendingEvents, isLoading: loadingEvents } =
    useEvents({ status: 'pending' });

  const pendingRemindersCount = pendingReminders?.length ?? 0;
  const pendingEventsCount = pendingEvents?.length ?? 0;
  const totalPending = pendingRemindersCount + pendingEventsCount;
  const hasPendingReminders = pendingRemindersCount > 0;
  const hasPendingEvents = pendingEventsCount > 0;

  const isInitialLoading =
    loadingReminders &&
    loadingEvents &&
    !pendingReminders &&
    !pendingEvents;

  const onRefresh = React.useCallback(async () => {
    setRefreshing(true);
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['events'] }),
      queryClient.invalidateQueries({ queryKey: ['reminders'] }),
      queryClient.invalidateQueries({ queryKey: ['appStatus'] }),
    ]);
    setRefreshing(false);
  }, [queryClient]);

  const jumpTo = React.useCallback((y: number) => {
    scrollViewRef.current?.scrollTo({
      y: Math.max(y - 12, 0),
      animated: true,
    });
  }, []);

  const jumpToReminders = React.useCallback(() => {
    jumpTo(remindersY);
  }, [jumpTo, remindersY]);

  const jumpToEvents = React.useCallback(() => {
    jumpTo(eventsY);
  }, [jumpTo, eventsY]);

  const handlePrimaryReviewAction = React.useCallback(() => {
    if (hasPendingReminders) {
      jumpToReminders();
      return;
    }

    if (hasPendingEvents) {
      jumpToEvents();
    }
  }, [hasPendingReminders, hasPendingEvents, jumpToReminders, jumpToEvents]);

  const handleRemindersLayout = React.useCallback((event: LayoutChangeEvent) => {
    setRemindersY(event.nativeEvent.layout.y);
  }, []);

  const handleEventsLayout = React.useCallback((event: LayoutChangeEvent) => {
    setEventsY(event.nativeEvent.layout.y);
  }, []);

  if (isInitialLoading) {
    return (
      <SafeAreaView style={styles.container} edges={['left', 'right']}>
        <View style={styles.loadingContainer}>
          <LoadingSpinner />
          <Text style={styles.loadingText}>Loading pending items...</Text>
        </View>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={styles.container} edges={['left', 'right']}>
      <ScrollView
        ref={scrollViewRef}
        style={styles.scrollView}
        contentContainerStyle={styles.content}
        showsVerticalScrollIndicator={false}
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            onRefresh={onRefresh}
            colors={[colors.primary]}
            tintColor={colors.primary}
          />
        }
      >
        <Card style={styles.heroCard}>
          <View style={styles.heroHeader}>
            <Ionicons
              name={totalPending > 0 ? 'alert-circle-outline' : 'checkmark-circle-outline'}
              size={18}
              color={totalPending > 0 ? colors.warning : colors.success}
            />
            <Text style={styles.heroLabel}>Needs Review</Text>
          </View>

          {totalPending > 0 ? (
            <>
              <Text style={styles.heroTitle}>
                {totalPending} pending item{totalPending === 1 ? '' : 's'} waiting for your decision
              </Text>
              <Text style={styles.heroDescription}>
                Review each suggestion below and accept, edit, or decline so Alfred can keep your plans accurate.
              </Text>

              <View style={styles.summaryRow}>
                <TouchableOpacity
                  activeOpacity={0.75}
                  style={[
                    styles.summaryChip,
                    !hasPendingReminders && styles.summaryChipDisabled,
                  ]}
                  disabled={!hasPendingReminders}
                  onPress={jumpToReminders}
                >
                  <View style={styles.summaryTopRow}>
                    <Text style={styles.summaryValue}>{pendingRemindersCount}</Text>
                    <Ionicons
                      name="chevron-forward"
                      size={14}
                      color={hasPendingReminders ? colors.textSecondary : colors.textMuted}
                    />
                  </View>
                  <Text style={styles.summaryLabel}>Reminders</Text>
                </TouchableOpacity>
                <TouchableOpacity
                  activeOpacity={0.75}
                  style={[
                    styles.summaryChip,
                    !hasPendingEvents && styles.summaryChipDisabled,
                  ]}
                  disabled={!hasPendingEvents}
                  onPress={jumpToEvents}
                >
                  <View style={styles.summaryTopRow}>
                    <Text style={styles.summaryValue}>{pendingEventsCount}</Text>
                    <Ionicons
                      name="chevron-forward"
                      size={14}
                      color={hasPendingEvents ? colors.textSecondary : colors.textMuted}
                    />
                  </View>
                  <Text style={styles.summaryLabel}>Events</Text>
                </TouchableOpacity>
              </View>

              <View style={styles.jumpRow}>
                <TouchableOpacity
                  activeOpacity={0.75}
                  style={[
                    styles.jumpButton,
                    !hasPendingReminders && styles.jumpButtonDisabled,
                  ]}
                  disabled={!hasPendingReminders}
                  onPress={jumpToReminders}
                >
                  <Ionicons
                    name="alarm-outline"
                    size={14}
                    color={hasPendingReminders ? colors.primary : colors.textMuted}
                    style={styles.jumpButtonIcon}
                  />
                  <Text
                    style={[
                      styles.jumpButtonText,
                      !hasPendingReminders && styles.jumpButtonTextDisabled,
                    ]}
                  >
                    Review reminders
                  </Text>
                </TouchableOpacity>
                <TouchableOpacity
                  activeOpacity={0.75}
                  style={[
                    styles.jumpButton,
                    !hasPendingEvents && styles.jumpButtonDisabled,
                  ]}
                  disabled={!hasPendingEvents}
                  onPress={jumpToEvents}
                >
                  <Ionicons
                    name="calendar-outline"
                    size={14}
                    color={hasPendingEvents ? colors.primary : colors.textMuted}
                    style={styles.jumpButtonIcon}
                  />
                  <Text
                    style={[
                      styles.jumpButtonText,
                      !hasPendingEvents && styles.jumpButtonTextDisabled,
                    ]}
                  >
                    Review events
                  </Text>
                </TouchableOpacity>
              </View>
            </>
          ) : (
            <>
              <Text style={styles.heroTitle}>All caught up</Text>
              <Text style={styles.heroDescription}>
                There are no pending reminders or events to review right now.
              </Text>
              <Button
                title="Back to Home"
                onPress={() => navigation.goBack()}
                style={styles.backHomeButton}
              />
            </>
          )}
        </Card>

        {totalPending > 0 && (
          <>
            <View onLayout={handleRemindersLayout}>
              <PendingRemindersSection />
            </View>
            <View onLayout={handleEventsLayout}>
              <PendingEventsSection compact={false} />
            </View>
          </>
        )}
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.background,
  },
  scrollView: {
    flex: 1,
  },
  content: {
    paddingHorizontal: 16,
    paddingTop: 0,
    paddingBottom: 28,
  },
  loadingContainer: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    gap: 10,
  },
  loadingText: {
    fontSize: 14,
    color: colors.textSecondary,
  },
  heroCard: {
    borderRadius: 14,
    borderWidth: 1,
    borderColor: colors.primary + '20',
    backgroundColor: '#f7fbff',
    paddingVertical: 14,
    paddingHorizontal: 14,
    marginBottom: 14,
  },
  heroHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 8,
  },
  heroLabel: {
    marginLeft: 8,
    fontSize: 12,
    fontWeight: '700',
    color: colors.primary,
    letterSpacing: 0.5,
    textTransform: 'uppercase',
  },
  heroTitle: {
    fontSize: 21,
    fontWeight: '700',
    color: colors.text,
    marginBottom: 6,
  },
  heroDescription: {
    fontSize: 13,
    color: colors.textSecondary,
    lineHeight: 18,
    marginBottom: 12,
  },
  primaryReviewButton: {
    marginBottom: 6,
  },
  primaryActionHint: {
    fontSize: 12,
    color: colors.textSecondary,
    marginBottom: 10,
  },
  summaryRow: {
    flexDirection: 'row',
    gap: 8,
    marginBottom: 10,
  },
  summaryChip: {
    flex: 1,
    backgroundColor: colors.card,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 10,
    paddingVertical: 10,
    paddingHorizontal: 10,
  },
  summaryChipDisabled: {
    backgroundColor: colors.background,
  },
  summaryTopRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  summaryValue: {
    fontSize: 20,
    fontWeight: '700',
    color: colors.text,
  },
  summaryLabel: {
    fontSize: 12,
    color: colors.textSecondary,
    marginTop: 2,
  },
  jumpRow: {
    flexDirection: 'row',
    gap: 8,
  },
  jumpButton: {
    flex: 1,
    borderRadius: 10,
    borderWidth: 1,
    borderColor: colors.primary + '50',
    backgroundColor: colors.primary + '10',
    paddingVertical: 10,
    alignItems: 'center',
    justifyContent: 'center',
    flexDirection: 'row',
  },
  jumpButtonDisabled: {
    borderColor: colors.border,
    backgroundColor: colors.card,
  },
  jumpButtonIcon: {
    marginRight: 6,
  },
  jumpButtonText: {
    fontSize: 12,
    fontWeight: '600',
    color: colors.primary,
  },
  jumpButtonTextDisabled: {
    color: colors.textMuted,
  },
  backHomeButton: {
    marginTop: 4,
  },
  guideCard: {
    marginBottom: 14,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 14,
    paddingVertical: 12,
    paddingHorizontal: 12,
  },
  guideTitle: {
    fontSize: 14,
    fontWeight: '700',
    color: colors.text,
    marginBottom: 10,
  },
  guideRow: {
    flexDirection: 'row',
    gap: 8,
  },
  guideItem: {
    flex: 1,
    backgroundColor: colors.background,
    borderRadius: 10,
    borderWidth: 1,
    borderColor: colors.border,
    paddingVertical: 10,
    paddingHorizontal: 8,
    alignItems: 'flex-start',
  },
  guideItemLabel: {
    marginTop: 4,
    fontSize: 12,
    fontWeight: '700',
    color: colors.text,
  },
  guideItemText: {
    marginTop: 2,
    fontSize: 11,
    color: colors.textSecondary,
    lineHeight: 15,
  },
});
