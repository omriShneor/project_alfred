import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { Ionicons } from '@expo/vector-icons';
import { Button } from '../../components/common';
import { colors } from '../../theme/colors';
import type { OnboardingParamList } from '../../navigation/OnboardingNavigator';

type NavigationProp = NativeStackNavigationProp<OnboardingParamList, 'Welcome'>;

export function WelcomeScreen() {
  const navigation = useNavigation<NavigationProp>();

  return (
    <SafeAreaView style={styles.container} edges={['top']}>
      <View style={styles.content}>
        <Text style={styles.appName}>ALFRED</Text>
        <Text style={styles.title}>Never miss what matters</Text>
        <Text style={styles.description}>
          Alfred finds events, reminders, and tasks in your chats and email service, then keeps your calendar and reminders up to date so nothing slips through.
        </Text>

        <View style={styles.timeBadge}>
          <Ionicons name="time-outline" size={14} color={colors.primary} />
          <Text style={styles.timeBadgeText}>Setup takes about 2 minutes</Text>
        </View>

        <View style={styles.features}>
          <View style={styles.featureItem}>
            <Ionicons name="apps-outline" size={16} color={colors.primary} />
            <Text style={styles.featureText}>Choose the apps you use every day</Text>
          </View>
          <View style={styles.featureItem}>
            <Ionicons name="shield-checkmark-outline" size={16} color={colors.primary} />
            <Text style={styles.featureText}>Approve each suggestion before it is saved</Text>
          </View>
          <View style={styles.featureItem}>
            <Ionicons name="calendar-outline" size={16} color={colors.primary} />
            <Text style={styles.featureText}>Sync approved events to your Google Calendar</Text>
          </View>
        </View>
      </View>

      <View style={styles.footer}>
        <Button
          title="Get Started"
          onPress={() => navigation.navigate('InputSelection')}
          style={styles.button}
        />
        <Text style={styles.footerHint}>Step 1: Choose your apps</Text>
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
    justifyContent: 'center',
  },
  appName: {
    fontSize: 14,
    fontWeight: '700',
    color: colors.primary,
    textTransform: 'uppercase',
    letterSpacing: 1,
    marginBottom: 8,
  },
  title: {
    fontSize: 34,
    fontWeight: '700',
    color: colors.text,
    marginBottom: 16,
    lineHeight: 40,
  },
  description: {
    fontSize: 16,
    color: colors.textSecondary,
    lineHeight: 24,
    marginBottom: 18,
  },
  timeBadge: {
    alignSelf: 'flex-start',
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 10,
    paddingVertical: 6,
    borderRadius: 999,
    backgroundColor: colors.primary + '14',
    marginBottom: 24,
  },
  timeBadgeText: {
    marginLeft: 6,
    fontSize: 12,
    fontWeight: '600',
    color: colors.primary,
  },
  features: {
    gap: 14,
  },
  featureItem: {
    flexDirection: 'row',
    alignItems: 'flex-start',
  },
  featureText: {
    fontSize: 15,
    color: colors.text,
    marginLeft: 10,
    flex: 1,
    lineHeight: 20,
  },
  footer: {
    paddingTop: 24,
    paddingBottom: 50,
  },
  button: {
    width: '100%',
  },
  footerHint: {
    marginTop: 10,
    textAlign: 'center',
    fontSize: 12,
    color: colors.textSecondary,
  },
});
