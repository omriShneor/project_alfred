import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { Button } from '../../components/common';
import { colors } from '../../theme/colors';
import type { OnboardingParamList } from '../../navigation/OnboardingNavigator';

type NavigationProp = NativeStackNavigationProp<OnboardingParamList, 'Welcome'>;

export function WelcomeScreen() {
  const navigation = useNavigation<NavigationProp>();

  return (
    <SafeAreaView style={styles.container} edges={['top']}>
      <View style={styles.content}>
        <Text style={styles.appName}>Alfred</Text>
        <Text style={styles.title}>Your Personal Assistant</Text>
        <Text style={styles.description}>
          Meet Alfred—your sidekick, helping you stay on top every single day
        </Text>

        <View style={styles.features}>
          <View style={styles.featureItem}>
            <Text style={styles.featureIcon}>*</Text>
            <Text style={styles.featureText}>Scan WhatsApp messages for events</Text>
          </View>
          <View style={styles.featureItem}>
            <Text style={styles.featureIcon}>*</Text>
            <Text style={styles.featureText}>Scan emails for appointments</Text>
          </View>
          <View style={styles.featureItem}>
            <Text style={styles.featureIcon}>*</Text>
            <Text style={styles.featureText}>Review before adding to calendar</Text>
          </View>
        </View>
      </View>

      <View style={styles.footer}>
        <Button
          title="Get Started"
          onPress={() => navigation.navigate('InputSelection')}
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
    justifyContent: 'center',
  },
  appName: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.primary,
    textTransform: 'uppercase',
    letterSpacing: 1,
    marginBottom: 8,
  },
  title: {
    fontSize: 32,
    fontWeight: 'bold',
    color: colors.text,
    marginBottom: 16,
  },
  description: {
    fontSize: 16,
    color: colors.textSecondary,
    lineHeight: 24,
    marginBottom: 32,
  },
  features: {
    gap: 16,
  },
  featureItem: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
  },
  featureIcon: {
    fontSize: 16,
    color: colors.primary,
    fontWeight: 'bold',
  },
  featureText: {
    fontSize: 15,
    color: colors.text,
  },
  footer: {
    paddingTop: 24,
  },
  button: {
    width: '100%',
  },
});
