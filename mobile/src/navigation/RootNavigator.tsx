import React, { useState, useEffect, useCallback } from 'react';
import { View, Text, StyleSheet, Linking } from 'react-native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import * as ExpoLinking from 'expo-linking';
import { useOnboardingStatus, useExchangeOAuthCode } from '../hooks';
import {
  WelcomeScreen,
  WhatsAppSetupScreen,
  GoogleCalendarSetupScreen,
  NotificationSetupScreen,
} from '../screens/onboarding';
import { TopTabs } from './TopTabs';
import { colors } from '../theme/colors';
import { LoadingSpinner } from '../components/common';

export type OnboardingStackParamList = {
  Welcome: undefined;
  WhatsAppSetup: undefined;
  GoogleCalendarSetup: undefined;
  NotificationSetup: undefined;
};

const OnboardingStack = createNativeStackNavigator<OnboardingStackParamList>();

function OnboardingNavigator({ onComplete }: { onComplete: () => void }) {
  return (
    <OnboardingStack.Navigator
      screenOptions={{
        headerStyle: { backgroundColor: colors.background },
        headerShadowVisible: false,
        headerTintColor: colors.text,
      }}
    >
      <OnboardingStack.Screen
        name="Welcome"
        component={WelcomeScreen}
        options={{ headerShown: false }}
      />
      <OnboardingStack.Screen
        name="WhatsAppSetup"
        component={WhatsAppSetupScreen}
        options={{ title: 'WhatsApp Setup', headerBackTitle: 'Back' }}
      />
      <OnboardingStack.Screen
        name="GoogleCalendarSetup"
        component={GoogleCalendarSetupScreen}
        options={{ title: 'Google Calendar', headerBackTitle: 'Back' }}
      />
      <OnboardingStack.Screen
        name="NotificationSetup"
        options={{ title: 'Notifications', headerBackTitle: 'Back' }}
      >
        {(props) => <NotificationSetupScreen {...props} onComplete={onComplete} />}
      </OnboardingStack.Screen>
    </OnboardingStack.Navigator>
  );
}

export function RootNavigator() {
  const [onboardingComplete, setOnboardingComplete] = useState<boolean | null>(null);
  const { data: status, isLoading, isError } = useOnboardingStatus();
  const exchangeCode = useExchangeOAuthCode();

  // Check if onboarding should be shown
  useEffect(() => {
    if (status) {
      // Show main app if at least one service is connected
      const hasAnyConnection =
        status.whatsapp.status === 'connected' ||
        status.gcal.status === 'connected';

      // Or if user has completed onboarding before (stored locally)
      // For now, just check connection status
      setOnboardingComplete(hasAnyConnection);
    }
  }, [status]);

  // Handle OAuth callback deep link globally
  const handleOAuthCallback = useCallback(
    async (code: string) => {
      const redirectUri = ExpoLinking.createURL('oauth/callback');
      try {
        await exchangeCode.mutateAsync({ code, redirectUri });
      } catch (error) {
        console.error('Failed to exchange OAuth code:', error);
      }
    },
    [exchangeCode]
  );

  // Listen for deep links
  useEffect(() => {
    const handleUrl = ({ url }: { url: string }) => {
      const parsed = ExpoLinking.parse(url);
      if (parsed.path === 'oauth/callback' && parsed.queryParams?.code) {
        handleOAuthCallback(parsed.queryParams.code as string);
      }
    };

    // Check initial URL
    ExpoLinking.getInitialURL().then((url) => {
      if (url) handleUrl({ url });
    });

    // Listen for new URLs
    const subscription = Linking.addEventListener('url', handleUrl);
    return () => subscription.remove();
  }, [handleOAuthCallback]);

  const handleOnboardingComplete = () => {
    setOnboardingComplete(true);
  };

  // Show loading state while checking status
  if (isLoading || onboardingComplete === null) {
    return (
      <View style={styles.loadingContainer}>
        <LoadingSpinner />
        <Text style={styles.loadingText}>Loading...</Text>
      </View>
    );
  }

  // Show error state if can't connect to backend
  if (isError) {
    return (
      <View style={styles.errorContainer}>
        <Text style={styles.errorIcon}>⚠️</Text>
        <Text style={styles.errorTitle}>Connection Error</Text>
        <Text style={styles.errorText}>
          Cannot connect to Alfred backend. Make sure the server is running.
        </Text>
      </View>
    );
  }

  // Show onboarding or main app
  if (!onboardingComplete) {
    return <OnboardingNavigator onComplete={handleOnboardingComplete} />;
  }

  return <TopTabs />;
}

const styles = StyleSheet.create({
  loadingContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: colors.background,
  },
  loadingText: {
    marginTop: 16,
    color: colors.textSecondary,
    fontSize: 16,
  },
  errorContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: colors.background,
    padding: 24,
  },
  errorIcon: {
    fontSize: 48,
    marginBottom: 16,
  },
  errorTitle: {
    fontSize: 20,
    fontWeight: 'bold',
    color: colors.text,
    marginBottom: 8,
  },
  errorText: {
    fontSize: 14,
    color: colors.textSecondary,
    textAlign: 'center',
    lineHeight: 20,
  },
});
