import React, { useState, useCallback } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ActivityIndicator, Image } from 'react-native';
import * as WebBrowser from 'expo-web-browser';
import { useAuth } from '../auth';
import { colors } from '../theme/colors';
import { requestLogin } from '../api/auth';

// Warm up the browser for faster OAuth redirect
WebBrowser.maybeCompleteAuthSession();

// Official Google "G" logo as PNG data URI (follows Google's branding guidelines)

export function LoginScreen() {
  const { login, isLoading } = useAuth();
  const [error, setError] = useState<string | null>(null);
  const [isSigningIn, setIsSigningIn] = useState(false);

  const handleGoogleSignIn = useCallback(async () => {
    setError(null);
    setIsSigningIn(true);

    try {
      console.log('[LoginScreen] Starting Google OAuth login...');

      // Use the new unified login endpoint (profile scopes only)
      const response = await requestLogin();

      console.log('[LoginScreen] Got auth URL, opening browser...');
      const result = await WebBrowser.openAuthSessionAsync(response.auth_url);
      console.log('[LoginScreen] Browser session completed:', result);

      if (result.type === 'success' && result.url) {
        const url = result.url;
        console.log('[LoginScreen] Got success URL:', url);

        // Extract code from URL
        const codeMatch = url.match(/[?&]code=([^&]+)/);
        if (codeMatch && codeMatch[1]) {
          const code = decodeURIComponent(codeMatch[1]);
          console.log('[LoginScreen] Extracted code, logging in...');

          // Exchange code and create session via the login callback
          await login(code, '');
          console.log('[LoginScreen] Login successful!');
        } else {
          throw new Error('No authorization code received');
        }
      } else if (result.type === 'cancel') {
        // User cancelled, no error
        console.log('[LoginScreen] User cancelled OAuth');
      } else {
        throw new Error('Authentication was not completed');
      }
    } catch (err) {
      console.error('[LoginScreen] Sign in error:', err);
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
        <Text style={styles.tagline}>Your Personal Virtual Assistant</Text>
      </View>

      <View style={styles.content}>
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
              <Image
                source={require('../../assets/google-logo.png')}
                style={styles.googleLogo}
              />
              <Text style={styles.googleButtonText}>Sign in with Google</Text>
            </>
          )}
        </TouchableOpacity>
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
    paddingTop: 200,
    alignItems: 'center',
  },
  logo: {
    fontSize: 48,
    fontWeight: '700',
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
    fontWeight: '700',
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
  googleLogo: {
    width: 20,
    height: 20,
    marginRight: 12,
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
