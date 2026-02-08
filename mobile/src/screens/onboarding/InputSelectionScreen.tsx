import React, { useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, Alert } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { Ionicons } from '@expo/vector-icons';
import { Button, Card } from '../../components/common';
import { colors } from '../../theme/colors';
import type { OnboardingParamList } from '../../navigation/OnboardingNavigator';

type NavigationProp = NativeStackNavigationProp<OnboardingParamList, 'InputSelection'>;

interface InputOption {
  id: 'whatsapp' | 'telegram' | 'gmail';
  title: string;
  description: string;
  icon: keyof typeof Ionicons.glyphMap;
}

const inputOptions: InputOption[] = [
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

export function InputSelectionScreen() {
  const navigation = useNavigation<NavigationProp>();
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const selectedCount = selected.size;
  const totalOptions = inputOptions.length;
  const progress = (selectedCount / totalOptions) * 100;
  const continueTitle =
    selectedCount > 0
      ? `Continue (${selectedCount} selected)`
      : 'Select at least one app';

  const toggleSelection = (id: string) => {
    const newSelected = new Set(selected);
    if (newSelected.has(id)) {
      newSelected.delete(id);
    } else {
      newSelected.add(id);
    }
    setSelected(newSelected);
  };

  const handleContinue = () => {
    if (selected.size === 0) {
      Alert.alert('Select at least one', 'Please select at least one app to continue.');
      return;
    }

    navigation.navigate('Connection', {
      whatsappEnabled: selected.has('whatsapp'),
      telegramEnabled: selected.has('telegram'),
      gmailEnabled: selected.has('gmail'),
    });
  };

  return (
    <SafeAreaView style={styles.container} edges={['top']}>
      <View style={styles.content}>
        <Text style={styles.step}>Step 1 of 3</Text>
        <Text style={styles.title}>Choose Your Apps</Text>
        <Text style={styles.description}>
          Select which apps Alfred should use. You can change this anytime.
        </Text>

        <Card style={styles.summaryCard}>
          <View style={styles.summaryRow}>
            <Text style={styles.summaryTitle}>
              {selectedCount} of {totalOptions} apps selected
            </Text>
            {selectedCount > 0 && (
              <TouchableOpacity onPress={() => setSelected(new Set())} style={styles.clearButton}>
                <Text style={styles.clearButtonText}>Clear</Text>
              </TouchableOpacity>
            )}
          </View>
          <View style={styles.progressTrack}>
            <View style={[styles.progressFill, { width: `${progress}%` }]} />
          </View>
          <Text style={styles.summaryText}>
            {selectedCount === 0
              ? 'Select at least one app to continue.'
              : 'Next, you will connect each selected app.'}
          </Text>
        </Card>

        <View style={styles.options}>
          {inputOptions.map((option) => {
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
          })}
        </View>
      </View>

      <View style={styles.footer}>
        <Button
          title={continueTitle}
          onPress={handleContinue}
          disabled={selected.size === 0}
          style={styles.button}
        />
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.background,
    padding: 24,
  },
  content: {
    flex: 1,
  },
  step: {
    fontSize: 14,
    color: colors.primary,
    fontWeight: '600',
    marginBottom: 8,
  },
  title: {
    fontSize: 28,
    fontWeight: '700',
    color: colors.text,
    marginBottom: 12,
  },
  description: {
    fontSize: 15,
    color: colors.textSecondary,
    lineHeight: 22,
    marginBottom: 16,
  },
  summaryCard: {
    marginBottom: 16,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: colors.primary + '20',
    backgroundColor: colors.infoBackground,
  },
  summaryRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 10,
  },
  summaryTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.text,
  },
  clearButton: {
    paddingHorizontal: 8,
    paddingVertical: 4,
  },
  clearButtonText: {
    fontSize: 12,
    fontWeight: '600',
    color: colors.primary,
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
  summaryText: {
    marginTop: 8,
    fontSize: 12,
    color: colors.textSecondary,
  },
  options: {
    gap: 12,
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
  footer: {
    paddingTop: 24,
    paddingBottom: 50,
  },
  button: {
    width: '100%',
  },
});
