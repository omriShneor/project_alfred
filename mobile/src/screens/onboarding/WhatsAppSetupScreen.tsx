import React, { useState, useEffect } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TextInput,
  ScrollView,
  Alert,
} from 'react-native';
import { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { Button, Card, LoadingSpinner } from '../../components/common';
import { colors } from '../../theme/colors';
import { useGeneratePairingCode, useWhatsAppStatus } from '../../hooks';

type OnboardingStackParamList = {
  Welcome: undefined;
  WhatsAppSetup: undefined;
  GoogleCalendarSetup: undefined;
  NotificationSetup: undefined;
};

interface Props {
  navigation: NativeStackNavigationProp<OnboardingStackParamList, 'WhatsAppSetup'>;
}

export function WhatsAppSetupScreen({ navigation }: Props) {
  const [phoneNumber, setPhoneNumber] = useState('');
  const [pairingCode, setPairingCode] = useState<string | null>(null);

  const { data: status, isLoading: statusLoading } = useWhatsAppStatus();
  const generateCode = useGeneratePairingCode();

  // Navigate to next step when connected
  useEffect(() => {
    if (status?.connected) {
      navigation.navigate('GoogleCalendarSetup');
    }
  }, [status?.connected, navigation]);

  const handleGenerateCode = async () => {
    if (!phoneNumber.trim()) {
      Alert.alert('Error', 'Please enter your phone number');
      return;
    }

    // Add + prefix if not present
    const formattedNumber = phoneNumber.startsWith('+')
      ? phoneNumber
      : `+${phoneNumber}`;

    try {
      const result = await generateCode.mutateAsync(formattedNumber);
      setPairingCode(result.code);
    } catch (error: any) {
      Alert.alert(
        'Error',
        error.response?.data?.error || 'Failed to generate pairing code'
      );
    }
  };

  if (statusLoading) {
    return (
      <View style={styles.loadingContainer}>
        <LoadingSpinner />
        <Text style={styles.loadingText}>Checking WhatsApp status...</Text>
      </View>
    );
  }

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <View style={styles.header}>
        <Text style={styles.stepIndicator}>Step 1 of 3</Text>
        <Text style={styles.title}>Connect WhatsApp</Text>
        <Text style={styles.subtitle}>
          Link your WhatsApp to monitor messages for calendar events
        </Text>
      </View>

      {status?.connected ? (
        <Card style={styles.successCard}>
          <Text style={styles.successIcon}>✓</Text>
          <Text style={styles.successText}>WhatsApp Connected</Text>
          <Button
            title="Continue"
            onPress={() => navigation.navigate('GoogleCalendarSetup')}
            style={styles.continueButton}
          />
        </Card>
      ) : pairingCode ? (
        <View>
          <Card style={styles.codeCard}>
            <Text style={styles.codeLabel}>Your Pairing Code</Text>
            <Text style={styles.code}>{pairingCode}</Text>
            <Text style={styles.codeExpiry}>
              Code expires in 5 minutes
            </Text>
          </Card>

          <Card style={styles.instructionsCard}>
            <Text style={styles.instructionsTitle}>How to link:</Text>
            <View style={styles.instructionsList}>
              <InstructionStep number={1} text="Open WhatsApp on your phone" />
              <InstructionStep number={2} text="Go to Settings > Linked Devices" />
              <InstructionStep number={3} text="Tap 'Link a Device'" />
              <InstructionStep number={4} text="Select 'Link with phone number instead'" />
              <InstructionStep number={5} text="Enter the code shown above" />
            </View>
          </Card>

          <View style={styles.waitingContainer}>
            <LoadingSpinner size="small" />
            <Text style={styles.waitingText}>
              Waiting for WhatsApp connection...
            </Text>
          </View>

          <Button
            title="Generate New Code"
            onPress={handleGenerateCode}
            variant="outline"
            loading={generateCode.isPending}
            style={styles.newCodeButton}
          />
        </View>
      ) : (
        <View>
          <Card>
            <Text style={styles.inputLabel}>Phone Number</Text>
            <Text style={styles.inputHint}>
              Enter your phone number with country code (e.g., +1234567890)
            </Text>
            <TextInput
              style={styles.input}
              value={phoneNumber}
              onChangeText={setPhoneNumber}
              placeholder="+1234567890"
              keyboardType="phone-pad"
              autoComplete="tel"
              autoFocus
            />
          </Card>

          <Button
            title="Generate Pairing Code"
            onPress={handleGenerateCode}
            loading={generateCode.isPending}
            disabled={!phoneNumber.trim()}
            size="large"
            style={styles.generateButton}
          />

          <Card style={styles.infoCard}>
            <Text style={styles.infoTitle}>Why pairing code?</Text>
            <Text style={styles.infoText}>
              Pairing codes allow you to link WhatsApp without scanning a QR
              code, which is convenient when setting up from your phone.
            </Text>
          </Card>
        </View>
      )}

      {!status?.connected && !pairingCode && (
        <Button
          title="Skip for now"
          onPress={() => navigation.navigate('GoogleCalendarSetup')}
          variant="outline"
          style={styles.skipButton}
        />
      )}
    </ScrollView>
  );
}

function InstructionStep({ number, text }: { number: number; text: string }) {
  return (
    <View style={styles.instructionStep}>
      <View style={styles.stepNumber}>
        <Text style={styles.stepNumberText}>{number}</Text>
      </View>
      <Text style={styles.stepText}>{text}</Text>
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
  codeCard: {
    alignItems: 'center',
    padding: 24,
    backgroundColor: colors.primary,
  },
  codeLabel: {
    fontSize: 14,
    color: 'rgba(255,255,255,0.8)',
    marginBottom: 8,
  },
  code: {
    fontSize: 36,
    fontWeight: 'bold',
    color: '#ffffff',
    letterSpacing: 4,
    fontFamily: 'monospace',
  },
  codeExpiry: {
    fontSize: 12,
    color: 'rgba(255,255,255,0.6)',
    marginTop: 8,
  },
  instructionsCard: {
    marginTop: 16,
  },
  instructionsTitle: {
    fontSize: 16,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 16,
  },
  instructionsList: {
    gap: 12,
  },
  instructionStep: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  stepNumber: {
    width: 24,
    height: 24,
    borderRadius: 12,
    backgroundColor: colors.primary,
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 12,
  },
  stepNumberText: {
    fontSize: 12,
    fontWeight: 'bold',
    color: '#ffffff',
  },
  stepText: {
    flex: 1,
    fontSize: 14,
    color: colors.text,
  },
  waitingContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: 24,
    marginBottom: 16,
  },
  waitingText: {
    marginLeft: 12,
    color: colors.textSecondary,
    fontSize: 14,
  },
  newCodeButton: {
    marginTop: 8,
  },
  inputLabel: {
    fontSize: 16,
    fontWeight: '600',
    color: colors.text,
    marginBottom: 4,
  },
  inputHint: {
    fontSize: 12,
    color: colors.textSecondary,
    marginBottom: 12,
  },
  input: {
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 8,
    padding: 12,
    fontSize: 18,
    color: colors.text,
    backgroundColor: colors.background,
  },
  generateButton: {
    marginTop: 16,
  },
  infoCard: {
    marginTop: 24,
    backgroundColor: '#e8f4fd',
  },
  infoTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.primary,
    marginBottom: 8,
  },
  infoText: {
    fontSize: 13,
    color: colors.text,
    lineHeight: 18,
  },
  skipButton: {
    marginTop: 24,
  },
});
