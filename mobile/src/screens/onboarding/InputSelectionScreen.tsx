import React, { useMemo, useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, Alert, ScrollView } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { Ionicons } from '@expo/vector-icons';
import { Button, Card } from '../../components/common';
import { colors } from '../../theme/colors';
import type { OnboardingParamList } from '../../navigation/OnboardingNavigator';

type NavigationProp = NativeStackNavigationProp<OnboardingParamList, 'InputSelection'>;
type InputOptionId = 'whatsapp' | 'telegram' | 'gmail' | 'gcal';

interface InputOption {
  id: InputOptionId;
  title: string;
  description: string;
  icon: keyof typeof Ionicons.glyphMap;
}

const dataSourceOptions: InputOption[] = [
  {
    id: 'whatsapp',
    title: 'WhatsApp',
    description: 'Detect reminders and events from selected chats',
    icon: 'chatbubble-outline',
  },
  {
    id: 'telegram',
    title: 'Telegram',
    description: 'Detect reminders and events from selected chats',
    icon: 'paper-plane-outline',
  },
  {
    id: 'gmail',
    title: 'Gmail',
    description: 'Detect reminders and events from selected senders',
    icon: 'mail-outline',
  },
];

const calendarOptions: InputOption[] = [
  {
    id: 'gcal',
    title: 'Google Calendar',
    description: 'Sync confirmed events to a calendar you choose',
    icon: 'calendar-outline',
  },
];

export function InputSelectionScreen() {
  const navigation = useNavigation<NavigationProp>();
  const [selected, setSelected] = useState<Set<InputOptionId>>(new Set<InputOptionId>());

  const selectedCount = selected.size;
  const requiredDataSourceCount = dataSourceOptions.length;
  const selectedDataSourceCount = useMemo(
    () => dataSourceOptions.filter((option) => selected.has(option.id)).length,
    [selected]
  );
  const canContinue = selectedDataSourceCount > 0;
  const progress = (selectedDataSourceCount / requiredDataSourceCount) * 100;
  const continueTitle = canContinue
    ? `Continue (${selectedDataSourceCount} data source${selectedDataSourceCount === 1 ? '' : 's'})`
    : 'Choose at least 1 data source';

  const heroHint = canContinue
    ? selected.has('gcal')
      ? 'Google Calendar sync will be connected separately in the next step.'
      : 'Google Calendar is optional. You can add it now or later.'
    : 'Pick at least one: Gmail, WhatsApp, or Telegram.';

  const toggleSelection = (id: InputOptionId) => {
    const nextSelected = new Set(selected);
    if (nextSelected.has(id)) {
      nextSelected.delete(id);
    } else {
      nextSelected.add(id);
    }
    setSelected(nextSelected);
  };

  const handleContinue = () => {
    if (!canContinue) {
      Alert.alert(
        'Select a data source',
        'Choose at least one data source: Gmail, Telegram, or WhatsApp.'
      );
      return;
    }

    navigation.navigate('Connection', {
      whatsappEnabled: selected.has('whatsapp'),
      telegramEnabled: selected.has('telegram'),
      gmailEnabled: selected.has('gmail'),
      gcalEnabled: selected.has('gcal'),
    });
  };

  const renderOption = (option: InputOption) => {
    const isSelected = selected.has(option.id);
    return (
      <TouchableOpacity
        key={option.id}
        style={[styles.option, isSelected && styles.optionSelected]}
        onPress={() => toggleSelection(option.id)}
        activeOpacity={0.7}
      >
        <View style={styles.optionContent}>
          <View style={[styles.iconContainer, isSelected && styles.iconContainerSelected]}>
            <Ionicons
              name={option.icon}
              size={24}
              color={isSelected ? colors.primary : colors.textSecondary}
            />
          </View>
          <View style={styles.optionText}>
            <Text style={styles.optionTitle}>{option.title}</Text>
            <Text style={styles.optionDescription}>{option.description}</Text>
          </View>
        </View>
        <View style={[styles.checkbox, isSelected && styles.checkboxSelected]}>
          {isSelected && <Ionicons name="checkmark" size={16} color="#fff" />}
        </View>
      </TouchableOpacity>
    );
  };

  return (
    <SafeAreaView style={styles.safeArea} edges={['top']}>
      <ScrollView
        style={styles.scrollView}
        contentContainerStyle={styles.content}
        keyboardShouldPersistTaps="handled"
        showsVerticalScrollIndicator={false}
      >
        <Card style={styles.heroCard}>
          <View style={styles.heroTopRow}>
            <Text style={styles.step}>Step 1 of 3</Text>
            <View
              style={[
                styles.heroStatusBadge,
                canContinue ? styles.heroStatusBadgeSuccess : styles.heroStatusBadgeWarning,
              ]}
            >
              <Text
                style={[
                  styles.heroStatusText,
                  canContinue ? styles.heroStatusTextSuccess : styles.heroStatusTextWarning,
                ]}
              >
                {selectedDataSourceCount}/{requiredDataSourceCount} sources
              </Text>
            </View>
          </View>
          <Text style={styles.title}>Choose Your Apps</Text>
          <Text style={styles.description}>
            Select at least one data source for Alfred. Google Calendar sync is optional.
          </Text>
          <View style={styles.progressTrack}>
            <View style={[styles.progressFill, { width: `${progress}%` }]} />
          </View>
          <Text style={styles.heroHint}>{heroHint}</Text>
          {selectedCount > 0 && (
            <TouchableOpacity
              onPress={() => setSelected(new Set<InputOptionId>())}
              style={styles.clearButton}
            >
              <Text style={styles.clearButtonText}>Clear selection</Text>
            </TouchableOpacity>
          )}
        </Card>

        <View style={styles.sectionHeader}>
          <Text style={styles.sectionTitle}>Data Sources</Text>
          <View style={[styles.sectionBadge, styles.sectionBadgeRequired]}>
            <Text style={[styles.sectionBadgeText, styles.sectionBadgeTextRequired]}>Required</Text>
          </View>
        </View>
        <View style={styles.options}>
          {dataSourceOptions.map(renderOption)}
        </View>

        <View style={styles.sectionHeader}>
          <Text style={styles.sectionTitle}>Calendar Sync</Text>
          <View style={[styles.sectionBadge, styles.sectionBadgeOptional]}>
            <Text style={[styles.sectionBadgeText, styles.sectionBadgeTextOptional]}>Optional</Text>
          </View>
        </View>
        <View style={styles.options}>
          {calendarOptions.map(renderOption)}
        </View>

        {!canContinue && (
          <View style={styles.ctaHint}>
            <Ionicons name="information-circle-outline" size={16} color={colors.warning} />
            <Text style={styles.ctaHintText}>
              Choose Gmail, Telegram, or WhatsApp to continue.
            </Text>
          </View>
        )}
      </ScrollView>

      <View style={styles.footer}>
        <Button
          title={continueTitle}
          onPress={handleContinue}
          disabled={!canContinue}
          style={styles.button}
        />
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: colors.background,
    padding: 24,
  },
  scrollView: {
    flex: 1,
  },
  content: {
    paddingBottom: 16,
  },
  heroCard: {
    borderRadius: 14,
    borderWidth: 1,
    borderColor: colors.primary + '22',
    backgroundColor: colors.infoBackground,
    marginBottom: 16,
  },
  heroTopRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 10,
  },
  step: {
    fontSize: 12,
    color: colors.primary,
    fontWeight: '700',
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  heroStatusBadge: {
    borderRadius: 999,
    borderWidth: 1,
    paddingHorizontal: 10,
    paddingVertical: 6,
  },
  heroStatusBadgeSuccess: {
    borderColor: colors.success + '45',
    backgroundColor: colors.success + '12',
  },
  heroStatusBadgeWarning: {
    borderColor: colors.warning + '45',
    backgroundColor: colors.warning + '12',
  },
  heroStatusText: {
    fontSize: 12,
    fontWeight: '700',
  },
  heroStatusTextSuccess: {
    color: colors.success,
  },
  heroStatusTextWarning: {
    color: colors.warning,
  },
  title: {
    fontSize: 28,
    fontWeight: '700',
    color: colors.text,
    marginBottom: 8,
  },
  description: {
    fontSize: 14,
    color: colors.textSecondary,
    lineHeight: 20,
    marginBottom: 12,
  },
  progressTrack: {
    height: 8,
    borderRadius: 999,
    backgroundColor: colors.border,
    overflow: 'hidden',
  },
  progressFill: {
    height: '100%',
    backgroundColor: colors.primary,
  },
  heroHint: {
    marginTop: 8,
    fontSize: 12,
    color: colors.textSecondary,
  },
  clearButton: {
    alignSelf: 'flex-start',
    marginTop: 10,
    paddingVertical: 4,
  },
  clearButtonText: {
    fontSize: 12,
    fontWeight: '600',
    color: colors.primary,
  },
  sectionHeader: {
    marginBottom: 8,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  sectionTitle: {
    fontSize: 14,
    fontWeight: '700',
    color: colors.text,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  sectionBadge: {
    borderWidth: 1,
    borderRadius: 999,
    paddingHorizontal: 10,
    paddingVertical: 4,
  },
  sectionBadgeRequired: {
    borderColor: colors.warning + '55',
    backgroundColor: colors.warning + '14',
  },
  sectionBadgeOptional: {
    borderColor: colors.textSecondary + '40',
    backgroundColor: colors.textSecondary + '12',
  },
  sectionBadgeText: {
    fontSize: 11,
    fontWeight: '700',
    letterSpacing: 0.4,
  },
  sectionBadgeTextRequired: {
    color: colors.warning,
  },
  sectionBadgeTextOptional: {
    color: colors.textSecondary,
  },
  options: {
    gap: 12,
    marginBottom: 16,
  },
  option: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: 16,
    borderRadius: 12,
    borderWidth: 2,
    borderColor: colors.border,
    backgroundColor: colors.card,
  },
  optionSelected: {
    borderColor: colors.primary,
    backgroundColor: `${colors.primary}10`,
  },
  optionContent: {
    flexDirection: 'row',
    alignItems: 'center',
    flex: 1,
  },
  iconContainer: {
    width: 48,
    height: 48,
    borderRadius: 24,
    backgroundColor: colors.background,
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 16,
  },
  iconContainerSelected: {
    backgroundColor: `${colors.primary}20`,
  },
  optionText: {
    flex: 1,
  },
  optionTitle: {
    fontSize: 17,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 2,
  },
  optionDescription: {
    fontSize: 13,
    color: colors.textSecondary,
    lineHeight: 18,
  },
  checkbox: {
    width: 24,
    height: 24,
    borderRadius: 12,
    borderWidth: 2,
    borderColor: colors.border,
    justifyContent: 'center',
    alignItems: 'center',
  },
  checkboxSelected: {
    backgroundColor: colors.primary,
    borderColor: colors.primary,
  },
  ctaHint: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 4,
  },
  ctaHintText: {
    marginLeft: 8,
    fontSize: 13,
    color: colors.textSecondary,
    fontWeight: '500',
  },
  footer: {
    paddingTop: 12,
    paddingBottom: 24,
  },
  button: {
    width: '100%',
  },
});
