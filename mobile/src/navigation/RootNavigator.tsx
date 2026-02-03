import React, { useEffect, useCallback } from 'react';
import { View, Text, StyleSheet, Linking } from 'react-native';
import * as ExpoLinking from 'expo-linking';
import { useExchangeOAuthCode, useHealth, useAppStatus } from '../hooks';
import { useAuth } from '../auth';
import { onAuthError } from '../api/client';
import { MainNavigator } from './MainNavigator';
import { OnboardingNavigator } from './OnboardingNavigator';
import { LoginScreen } from '../screens/LoginScreen';
import { colors } from '../theme/colors';
import { LoadingSpinner } from '../components/common';

export function RootNavigator() {
  const { isAuthenticated, isLoading: authLoading, login, logout } = useAuth();
  const { isLoading: healthLoading, isError: healthError } = useHealth();
  const { data: appStatus, isLoading: statusLoading, refetch: refetchAppStatus } = useAppStatus();
  const exchangeCode = useExchangeOAuthCode();

  // Listen for auth errors (401 responses) to handle session expiry
  useEffect(() => {
    const unsubscribe = onAuthError(() => {
      // Auth error occurred, user will be logged out automatically
      // The auth context will update isAuthenticated to false
    });
    return unsubscribe;
  }, []);

  // Handle OAuth callback deep link globally (for Google Calendar OAuth)
  const handleOAuthCallback = useCallback(
    async (code: string) => {
      const redirectUri = ExpoLinking.createURL('oauth/callback');
      try {
        // If this is a Google sign-in callback, use the auth login
        // Otherwise, it might be a Google Calendar OAuth callback
        if (!isAuthenticated) {
          await login(code, redirectUri);
        } else {
          // User already authenticated, this is likely a Google Calendar OAuth
          await exchangeCode.mutateAsync({ code, redirectUri });
        }
      } catch (error) {
        console.error('Failed to handle OAuth callback:', error);
      }
    },
    [exchangeCode, isAuthenticated, login]
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

  // Refetch app status when authentication changes
  useEffect(() => {
    if (isAuthenticated) {
      refetchAppStatus();
    }
  }, [isAuthenticated, refetchAppStatus]);

  // Show loading state while checking auth and backend health
  if (authLoading || healthLoading) {
    return (
      <View style={styles.loadingContainer}>
        <LoadingSpinner />
        <Text style={styles.loadingText}>Loading...</Text>
      </View>
    );
  }

  // Show error state if can't connect to backend
  if (healthError) {
    return (
      <View style={styles.errorContainer}>
        <Text style={styles.errorIcon}>!</Text>
        <Text style={styles.errorTitle}>Connection Error</Text>
        <Text style={styles.errorText}>
          Cannot connect to Alfred backend. Make sure the server is running.
        </Text>
      </View>
    );
  }

  // Show login screen if not authenticated
  if (!isAuthenticated) {
    return <LoginScreen />;
  }

  // Show loading while fetching app status after authentication
  if (statusLoading) {
    return (
      <View style={styles.loadingContainer}>
        <LoadingSpinner />
        <Text style={styles.loadingText}>Loading...</Text>
      </View>
    );
  }

  // Show onboarding if not completed
  if (!appStatus?.onboarding_complete) {
    return <OnboardingNavigator />;
  }

  // Show main app
  return <MainNavigator />;
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
    color: colors.warning,
    fontWeight: 'bold',
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
