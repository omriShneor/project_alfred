import React, { useEffect, useCallback } from 'react';
import { View, Text, StyleSheet, ScrollView, Alert, Linking } from 'react-native';
import { NativeStackNavigationProp } from '@react-navigation/native-stack';
import * as WebBrowser from 'expo-web-browser';
import * as ExpoLinking from 'expo-linking';
import { Button, Card, LoadingSpinner } from '../../components/common';
import { colors } from '../../theme/colors';
import { useGCalStatus, useGetOAuthURL, useExchangeOAuthCode } from '../../hooks';

type OnboardingStackParamList = {
  Welcome: undefined;
  WhatsAppSetup: undefined;
  GoogleCalendarSetup: undefined;
  NotificationSetup: undefined;
};

interface Props {
  navigation: NativeStackNavigationProp<OnboardingStackParamList, 'GoogleCalendarSetup'>;
}

// Warm up the browser for faster OAuth
WebBrowser.maybeCompleteAuthSession();

export function GoogleCalendarSetupScreen({ navigation }: Props) {
  const { data: status, isLoading: statusLoading, refetch } = useGCalStatus();
  const getOAuthURL = useGetOAuthURL();
  const exchangeCode = useExchangeOAuthCode();

  // Navigate to next step when connected
  useEffect(() => {
    if (status?.connected) {
      // Small delay to show success state
      const timeout = setTimeout(() => {
        navigation.navigate('NotificationSetup');
      }, 1500);
      return () => clearTimeout(timeout);
    }
  }, [status?.connected, navigation]);

  // Handle OAuth callback from deep link
  const handleOAuthCallback = useCallback(
    async (code: string) => {
      const redirectUri = ExpoLinking.createURL('oauth/callback');
      try {
        await exchangeCode.mutateAsync({ code, redirectUri });
        refetch();
      } catch (error: any) {
        Alert.alert(
          'Error',
          error.response?.data?.error || 'Failed to connect Google Calendar'
        );
      }
    },
    [exchangeCode, refetch]
  );

  // Listen for deep link callback
  useEffect(() => {
    const handleUrl = ({ url }: { url: string }) => {
      const parsed = ExpoLinking.parse(url);
      if (parsed.path === 'oauth/callback' && parsed.queryParams?.code) {
        handleOAuthCallback(parsed.queryParams.code as string);
      }
    };

    // Check if app was opened with a URL
    ExpoLinking.getInitialURL().then((url) => {
      if (url) handleUrl({ url });
    });

    // Listen for URL events
    const subscription = Linking.addEventListener('url', handleUrl);
    return () => subscription.remove();
  }, [handleOAuthCallback]);

  const handleConnectGoogle = async () => {
    const redirectUri = ExpoLinking.createURL('oauth/callback');

    try {
      const response = await getOAuthURL.mutateAsync(redirectUri);

      // Open browser for OAuth
      const result = await WebBrowser.openAuthSessionAsync(
        response.auth_url,
        redirectUri
      );

      if (result.type === 'success' && result.url) {
        const parsed = ExpoLinking.parse(result.url);
        if (parsed.queryParams?.code) {
          await handleOAuthCallback(parsed.queryParams.code as string);
        }
      } else if (result.type === 'cancel') {
        // User cancelled, that's OK
      }
    } catch (error: any) {
      Alert.alert(
        'Error',
        error.response?.data?.error || 'Failed to start Google authorization'
      );
    }
  };

  if (statusLoading) {
    return (
      <View style={styles.loadingContainer}>
        <LoadingSpinner />
        <Text style={styles.loadingText}>Checking Google Calendar status...</Text>
      </View>
    );
  }

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <View style={styles.header}>
        <Text style={styles.stepIndicator}>Step 2 of 3</Text>
        <Text style={styles.title}>Connect Google Calendar</Text>
        <Text style={styles.subtitle}>
          Allow Alfred to create and manage events in your calendar
        </Text>
      </View>

      {status?.connected ? (
        <Card style={styles.successCard}>
          <Text style={styles.successIcon}>✓</Text>
          <Text style={styles.successText}>Google Calendar Connected</Text>
          <Button
            title="Continue"
            onPress={() => navigation.navigate('NotificationSetup')}
            style={styles.continueButton}
          />
        </Card>
      ) : (
        <View>
          <Card style={styles.connectCard}>
            <View style={styles.googleIcon}>
              <Text style={styles.googleIconText}>G</Text>
            </View>
            <Text style={styles.connectTitle}>Google Calendar</Text>
            <Text style={styles.connectDescription}>
              Connect your Google account to sync detected events to your calendar
            </Text>
            <Button
              title="Connect Google Calendar"
              onPress={handleConnectGoogle}
              loading={getOAuthURL.isPending || exchangeCode.isPending}
              size="large"
              style={styles.connectButton}
            />
          </Card>

          <Card style={styles.permissionsCard}>
            <Text style={styles.permissionsTitle}>Permissions requested:</Text>
            <View style={styles.permissionsList}>
              <PermissionItem
                icon="📅"
                text="View and edit events on your calendars"
              />
              <PermissionItem
                icon="📋"
                text="See the list of your calendars"
              />
            </View>
            <Text style={styles.permissionsNote}>
              Alfred only accesses calendars you choose. Your data stays private.
            </Text>
          </Card>
        </View>
      )}

      {!status?.connected && (
        <Button
          title="Skip for now"
          onPress={() => navigation.navigate('NotificationSetup')}
          variant="outline"
          style={styles.skipButton}
        />
      )}
    </ScrollView>
  );
}

function PermissionItem({ icon, text }: { icon: string; text: string }) {
  return (
    <View style={styles.permissionItem}>
      <Text style={styles.permissionIcon}>{icon}</Text>
      <Text style={styles.permissionText}>{text}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.background,
  },
  content: {
    padding: 24,
  },
  loadingContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: colors.background,
  },
  loadingText: {
    marginTop: 16,
    color: colors.textSecondary,
  },
  header: {
    marginBottom: 24,
  },
  stepIndicator: {
    fontSize: 12,
    color: colors.primary,
    fontWeight: '600',
    marginBottom: 8,
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
    color: colors.text,
    marginBottom: 8,
  },
  subtitle: {
    fontSize: 14,
    color: colors.textSecondary,
    lineHeight: 20,
  },
  successCard: {
    alignItems: 'center',
    padding: 32,
  },
  successIcon: {
    fontSize: 48,
    color: colors.success,
    marginBottom: 16,
  },
  successText: {
    fontSize: 18,
    fontWeight: '600',
    color: colors.success,
    marginBottom: 24,
  },
  continueButton: {
    width: '100%',
  },
  connectCard: {
    alignItems: 'center',
    padding: 24,
  },
  googleIcon: {
    width: 64,
    height: 64,
    borderRadius: 32,
    backgroundColor: '#4285f4',
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 16,
  },
  googleIconText: {
    fontSize: 32,
    fontWeight: 'bold',
    color: '#ffffff',
  },
  connectTitle: {
    fontSize: 20,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 8,
  },
  connectDescription: {
    fontSize: 14,
    color: colors.textSecondary,
    textAlign: 'center',
    marginBottom: 24,
  },
  connectButton: {
    width: '100%',
  },
  permissionsCard: {
    marginTop: 16,
  },
  permissionsTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 12,
  },
  permissionsList: {
    gap: 8,
    marginBottom: 12,
  },
  permissionItem: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  permissionIcon: {
    fontSize: 16,
    marginRight: 12,
  },
  permissionText: {
    fontSize: 13,
    color: colors.text,
    flex: 1,
  },
  permissionsNote: {
    fontSize: 12,
    color: colors.textSecondary,
    fontStyle: 'italic',
  },
  skipButton: {
    marginTop: 24,
  },
});
