import React from 'react';
import { View, Text, StyleSheet, Image } from 'react-native';
import { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { Button } from '../../components/common';
import { colors } from '../../theme/colors';

type OnboardingStackParamList = {
  Welcome: undefined;
  WhatsAppSetup: undefined;
  GoogleCalendarSetup: undefined;
  NotificationSetup: undefined;
};

interface Props {
  navigation: NativeStackNavigationProp<OnboardingStackParamList, 'Welcome'>;
}

export function WelcomeScreen({ navigation }: Props) {
  return (
    <View style={styles.container}>
      <View style={styles.content}>
        <View style={styles.iconContainer}>
          <Text style={styles.icon}>🗓</Text>
        </View>
        <Text style={styles.title}>Welcome to Alfred</Text>
        <Text style={styles.subtitle}>
          Your WhatsApp-to-Calendar assistant
        </Text>
        <Text style={styles.description}>
          Alfred automatically detects calendar events from your WhatsApp
          messages and syncs them to Google Calendar.
        </Text>

        <View style={styles.features}>
          <FeatureItem
            icon="💬"
            title="WhatsApp Integration"
            description="Connect your WhatsApp to monitor messages"
          />
          <FeatureItem
            icon="🤖"
            title="AI Detection"
            description="Claude AI finds events in conversations"
          />
          <FeatureItem
            icon="📅"
            title="Calendar Sync"
            description="Review and sync events to Google Calendar"
          />
        </View>
      </View>

      <View style={styles.footer}>
        <Button
          title="Get Started"
          onPress={() => navigation.navigate('WhatsAppSetup')}
          size="large"
          style={styles.button}
        />
      </View>
    </View>
  );
}

function FeatureItem({
  icon,
  title,
  description,
}: {
  icon: string;
  title: string;
  description: string;
}) {
  return (
    <View style={styles.featureItem}>
      <Text style={styles.featureIcon}>{icon}</Text>
      <View style={styles.featureText}>
        <Text style={styles.featureTitle}>{title}</Text>
        <Text style={styles.featureDescription}>{description}</Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.background,
  },
  content: {
    flex: 1,
    padding: 24,
    justifyContent: 'center',
  },
  iconContainer: {
    alignItems: 'center',
    marginBottom: 16,
  },
  icon: {
    fontSize: 64,
  },
  title: {
    fontSize: 28,
    fontWeight: 'bold',
    color: colors.text,
    textAlign: 'center',
    marginBottom: 8,
  },
  subtitle: {
    fontSize: 16,
    color: colors.primary,
    textAlign: 'center',
    marginBottom: 16,
  },
  description: {
    fontSize: 14,
    color: colors.textSecondary,
    textAlign: 'center',
    lineHeight: 20,
    marginBottom: 32,
  },
  features: {
    gap: 16,
  },
  featureItem: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: colors.card,
    padding: 16,
    borderRadius: 12,
  },
  featureIcon: {
    fontSize: 28,
    marginRight: 16,
  },
  featureText: {
    flex: 1,
  },
  featureTitle: {
    fontSize: 16,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 2,
  },
  featureDescription: {
    fontSize: 13,
    color: colors.textSecondary,
  },
  footer: {
    padding: 24,
    paddingBottom: 40,
  },
  button: {
    width: '100%',
  },
});
