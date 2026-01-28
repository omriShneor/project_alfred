import React, { useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TouchableOpacity,
  Alert,
} from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { Feather } from '@expo/vector-icons';
import { Button, Card } from '../../components/common';
import { colors } from '../../theme/colors';
import { useUpdateSmartCalendar } from '../../hooks';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import type { SmartCalendarStackParamList } from '../../navigation/DrawerNavigator';

type NavigationProp = NativeStackNavigationProp<SmartCalendarStackParamList>;

interface CheckboxItemProps {
  label: string;
  description?: string;
  checked: boolean;
  onToggle: () => void;
  disabled?: boolean;
  comingSoon?: boolean;
}

function CheckboxItem({ label, description, checked, onToggle, disabled, comingSoon }: CheckboxItemProps) {
  return (
    <TouchableOpacity
      style={[styles.checkboxItem, disabled && styles.checkboxItemDisabled]}
      onPress={onToggle}
      disabled={disabled}
      activeOpacity={0.7}
    >
      <View style={[styles.checkbox, checked && styles.checkboxChecked]}>
        {checked && <Feather name="check" size={14} color="#fff" />}
      </View>
      <View style={styles.checkboxContent}>
        <View style={styles.checkboxLabelRow}>
          <Text style={[styles.checkboxLabel, disabled && styles.checkboxLabelDisabled]}>
            {label}
          </Text>
          {comingSoon && (
            <View style={styles.comingSoonBadge}>
              <Text style={styles.comingSoonText}>Coming soon</Text>
            </View>
          )}
        </View>
        {description && (
          <Text style={[styles.checkboxDescription, disabled && styles.checkboxDescriptionDisabled]}>
            {description}
          </Text>
        )}
      </View>
    </TouchableOpacity>
  );
}

export function SmartCalendarSetupScreen() {
  const navigation = useNavigation<NavigationProp>();
  const updateSmartCalendar = useUpdateSmartCalendar();

  // Input selections - default to unchecked
  const [whatsappEnabled, setWhatsappEnabled] = useState(false);
  const [emailEnabled, setEmailEnabled] = useState(false);
  const [smsEnabled, setSmsEnabled] = useState(false);

  // Calendar selections - Alfred is checked by default
  const [alfredCalendarEnabled, setAlfredCalendarEnabled] = useState(true);
  const [googleCalendarEnabled, setGoogleCalendarEnabled] = useState(false);
  const [outlookEnabled, setOutlookEnabled] = useState(false);

  const hasSelectedInput = whatsappEnabled || emailEnabled;
  const hasSelectedCalendar = alfredCalendarEnabled || googleCalendarEnabled;
  const canContinue = hasSelectedInput && hasSelectedCalendar;

  const handleContinue = async () => {
    if (!canContinue) {
      Alert.alert('Selection Required', 'Please select at least one input and one calendar');
      return;
    }

    try {
      // Save the selections and enable Smart Calendar
      await updateSmartCalendar.mutateAsync({
        enabled: true,
        inputs: {
          whatsapp: whatsappEnabled,
          email: emailEnabled,
          sms: smsEnabled,
        },
        calendars: {
          alfred: alfredCalendarEnabled,
          google_calendar: googleCalendarEnabled,
          outlook: outlookEnabled,
        },
      });

      // Navigate to permissions screen
      navigation.navigate('Permissions');
    } catch (error: any) {
      Alert.alert('Error', error.message || 'Failed to save settings');
    }
  };

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      {/* Inputs Section */}
      <Text style={styles.sectionTitle}>Inputs</Text>
      <Text style={styles.sectionDescription}>
        Where should we look for events? Messages from these sources will be scanned to detect calendar events.
      </Text>
      <Card style={styles.card}>
        <CheckboxItem
          label="WhatsApp"
          description="Scan WhatsApp messages for events"
          checked={whatsappEnabled}
          onToggle={() => setWhatsappEnabled(!whatsappEnabled)}
        />
        <View style={styles.divider} />
        <CheckboxItem
          label="Email (Gmail)"
          description="Scan Gmail inbox for events"
          checked={emailEnabled}
          onToggle={() => setEmailEnabled(!emailEnabled)}
        />
        <View style={styles.divider} />
        <CheckboxItem
          label="SMS"
          checked={smsEnabled}
          onToggle={() => {}}
          disabled
          comingSoon
        />
      </Card>

      {/* Calendars Section */}
      <Text style={styles.sectionTitle}>Calendars</Text>
      <Text style={styles.sectionDescription}>
        Where should we add events? Detected events will be synced to your selected calendars.
      </Text>
      <Card style={styles.card}>
        <CheckboxItem
          label="Alfred Calendar"
          description="Store events locally in Project Alfred"
          checked={alfredCalendarEnabled}
          onToggle={() => setAlfredCalendarEnabled(!alfredCalendarEnabled)}
        />
        <View style={styles.divider} />
        <CheckboxItem
          label="Google Calendar"
          description="Sync events to Google Calendar"
          checked={googleCalendarEnabled}
          onToggle={() => setGoogleCalendarEnabled(!googleCalendarEnabled)}
        />
        <View style={styles.divider} />
        <CheckboxItem
          label="Outlook"
          checked={outlookEnabled}
          onToggle={() => {}}
          disabled
          comingSoon
        />
      </Card>

      {/* Validation message */}
      {!canContinue && (
        <Text style={styles.validationMessage}>
          Select at least one input and one calendar to continue
        </Text>
      )}

      {/* Continue Button */}
      <Button
        title="Save & Continue"
        onPress={handleContinue}
        disabled={!canContinue}
        loading={updateSmartCalendar.isPending}
        style={styles.continueButton}
      />
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.background,
  },
  content: {
    padding: 16,
    paddingBottom: 32,
  },
  sectionTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.textSecondary,
    marginTop: 16,
    marginBottom: 8,
    marginLeft: 4,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  sectionDescription: {
    fontSize: 14,
    color: colors.textSecondary,
    marginBottom: 12,
    marginHorizontal: 4,
    lineHeight: 20,
  },
  card: {
    paddingVertical: 4,
  },
  checkboxItem: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 12,
    paddingHorizontal: 4,
  },
  checkboxItemDisabled: {
    opacity: 0.5,
  },
  checkbox: {
    width: 22,
    height: 22,
    borderRadius: 4,
    borderWidth: 2,
    borderColor: colors.border,
    alignItems: 'center',
    justifyContent: 'center',
    marginRight: 12,
  },
  checkboxChecked: {
    backgroundColor: colors.primary,
    borderColor: colors.primary,
  },
  checkboxContent: {
    flex: 1,
  },
  checkboxLabelRow: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  checkboxLabel: {
    fontSize: 16,
    fontWeight: '500',
    color: colors.text,
  },
  checkboxLabelDisabled: {
    color: colors.textSecondary,
  },
  checkboxDescription: {
    fontSize: 13,
    color: colors.textSecondary,
    marginTop: 2,
  },
  checkboxDescriptionDisabled: {
    color: colors.textSecondary,
  },
  comingSoonBadge: {
    backgroundColor: colors.border,
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: 4,
    marginLeft: 8,
  },
  comingSoonText: {
    fontSize: 10,
    color: colors.textSecondary,
    fontWeight: '600',
    textTransform: 'uppercase',
  },
  divider: {
    height: 1,
    backgroundColor: colors.border,
    marginLeft: 34,
  },
  validationMessage: {
    fontSize: 13,
    color: colors.warning,
    textAlign: 'center',
    marginTop: 16,
  },
  continueButton: {
    marginTop: 24,
  },
});
