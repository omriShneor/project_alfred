import React, { useEffect, useCallback } from 'react';
import { View, Text, StyleSheet, Linking } from 'react-native';
import * as ExpoLinking from 'expo-linking';
import { useExchangeOAuthCode, useHealth } from '../hooks';
import { DrawerNavigator } from './DrawerNavigator';
import { useAppState } from '../context/AppStateContext';
import { colors } from '../theme/colors';
import { LoadingSpinner } from '../components/common';

export function RootNavigator() {
  const { isLoading, isError } = useHealth();
  const exchangeCode = useExchangeOAuthCode();
  const { setShowDrawerToggle } = useAppState();

  // Always show drawer toggle since we're skipping mandatory onboarding
  useEffect(() => {
    setShowDrawerToggle(true);
  }, [setShowDrawerToggle]);

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

  // Show loading state while checking backend health
  if (isLoading) {
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
        <Text style={styles.errorIcon}>!</Text>
        <Text style={styles.errorTitle}>Connection Error</Text>
        <Text style={styles.errorText}>
          Cannot connect to Alfred backend. Make sure the server is running.
        </Text>
      </View>
    );
  }

  // Always show main app - no mandatory onboarding
  return <DrawerNavigator />;
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
