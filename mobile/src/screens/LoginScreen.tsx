import React, { useState, useCallback } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ActivityIndicator } from 'react-native';
import * as WebBrowser from 'expo-web-browser';
import * as ExpoLinking from 'expo-linking';
import { useAuth } from '../auth';
import { colors } from '../theme/colors';
import { API_BASE_URL } from '../config/api';

// Warm up the browser for faster OAuth redirect
WebBrowser.maybeCompleteAuthSession();

export function LoginScreen() {
  const { login, isLoading } = useAuth();
  const [error, setError] = useState<string | null>(null);
  const [isSigningIn, setIsSigningIn] = useState(false);

  const handleGoogleSignIn = useCallback(async () => {
    setError(null);
    setIsSigningIn(true);

    try {
      // Get the redirect URI for our app
      const redirectUri = ExpoLinking.createURL('oauth/callback');

      // Get the Google OAuth URL from our backend
      const response = await fetch(`${API_BASE_URL}/api/auth/google?redirect_uri=${encodeURIComponent(redirectUri)}`);

      if (!response.ok) {
        throw new Error('Failed to get authentication URL');
      }

      const { auth_url } = await response.json();

      // Open the browser for Google sign-in
      const result = await WebBrowser.openAuthSessionAsync(auth_url, redirectUri);

      if (result.type === 'success' && result.url) {
        // Extract the code from the callback URL
        const parsed = ExpoLinking.parse(result.url);
        const code = parsed.queryParams?.code as string | undefined;

        if (code) {
          await login(code, redirectUri);
        } else {
          throw new Error('No authorization code received');
        }
      } else if (result.type === 'cancel') {
        // User cancelled, no error
      } else {
        throw new Error('Authentication was not completed');
      }
    } catch (err) {
      console.error('Sign in error:', err);
      setError(err instanceof Error ? err.message : 'Sign in failed');
    } finally {
      setIsSigningIn(false);
    }
  }, [login]);

  const showLoading = isLoading || isSigningIn;

  return (
    <View style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.logo}>Alfred</Text>
        <Text style={styles.tagline}>Your Personal Calendar Assistant</Text>
      </View>

      <View style={styles.content}>
        <Text style={styles.title}>Welcome</Text>
        <Text style={styles.subtitle}>
          Sign in with your Google account to get started
        </Text>

        {error && (
          <View style={styles.errorContainer}>
            <Text style={styles.errorText}>{error}</Text>
          </View>
        )}

        <TouchableOpacity
          style={[styles.googleButton, showLoading && styles.googleButtonDisabled]}
          onPress={handleGoogleSignIn}
          disabled={showLoading}
          activeOpacity={0.8}
        >
          {showLoading ? (
            <ActivityIndicator color={colors.text} />
          ) : (
            <>
              <View style={styles.googleIcon}>
                <Text style={styles.googleIconText}>G</Text>
              </View>
              <Text style={styles.googleButtonText}>Sign in with Google</Text>
            </>
          )}
        </TouchableOpacity>
      </View>

      <View style={styles.footer}>
        <Text style={styles.footerText}>
          By signing in, you agree to allow Alfred to access your calendar
        </Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.background,
    padding: 24,
  },
  header: {
    paddingTop: 60,
    alignItems: 'center',
  },
  logo: {
    fontSize: 48,
    fontWeight: 'bold',
    color: colors.primary,
    letterSpacing: -1,
  },
  tagline: {
    fontSize: 16,
    color: colors.textSecondary,
    marginTop: 8,
  },
  content: {
    flex: 1,
    justifyContent: 'center',
    paddingHorizontal: 16,
  },
  title: {
    fontSize: 32,
    fontWeight: 'bold',
    color: colors.text,
    textAlign: 'center',
    marginBottom: 8,
  },
  subtitle: {
    fontSize: 16,
    color: colors.textSecondary,
    textAlign: 'center',
    marginBottom: 32,
    lineHeight: 24,
  },
  errorContainer: {
    backgroundColor: colors.dangerBackground,
    padding: 12,
    borderRadius: 8,
    marginBottom: 16,
  },
  errorText: {
    color: colors.danger,
    fontSize: 14,
    textAlign: 'center',
  },
  googleButton: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: colors.surface,
    paddingVertical: 16,
    paddingHorizontal: 24,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: colors.border,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.1,
    shadowRadius: 4,
    elevation: 3,
  },
  googleButtonDisabled: {
    opacity: 0.6,
  },
  googleIcon: {
    width: 24,
    height: 24,
    borderRadius: 12,
    backgroundColor: colors.primary,
    alignItems: 'center',
    justifyContent: 'center',
    marginRight: 12,
  },
  googleIconText: {
    color: '#fff',
    fontSize: 14,
    fontWeight: 'bold',
  },
  googleButtonText: {
    fontSize: 16,
    fontWeight: '600',
    color: colors.text,
  },
  footer: {
    paddingBottom: 32,
  },
  footerText: {
    fontSize: 12,
    color: colors.textMuted,
    textAlign: 'center',
    lineHeight: 18,
  },
});
