import React from 'react';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { WelcomeScreen } from '../screens/onboarding/WelcomeScreen';
import { InputSelectionScreen } from '../screens/onboarding/InputSelectionScreen';
import { ConnectionScreen } from '../screens/onboarding/ConnectionScreen';
import { SourceConfigurationScreen } from '../screens/onboarding/SourceConfigurationScreen';
import {
  WhatsAppPreferencesScreen,
  TelegramPreferencesScreen,
  GmailPreferencesScreen,
  GoogleCalendarPreferencesScreen,
} from '../screens/smart-calendar';
import { colors } from '../theme/colors';
import {
  createBackHeaderOptions,
  stackGestureBackOptions,
} from './sharedHeader';

export type OnboardingParamList = {
  Welcome: undefined;
  InputSelection: undefined;
  Connection: {
    whatsappEnabled: boolean;
    telegramEnabled: boolean;
    gmailEnabled: boolean;
    gcalEnabled: boolean;
  };
  SourceConfiguration: {
    whatsappEnabled: boolean;
    telegramEnabled: boolean;
    gmailEnabled: boolean;
    gcalEnabled: boolean;
  };
  // Add preference screens from shared stack
  WhatsAppPreferences: undefined;
  TelegramPreferences: undefined;
  GmailPreferences: undefined;
  GoogleCalendarPreferences: undefined;
};

const Stack = createNativeStackNavigator<OnboardingParamList>();

export function OnboardingNavigator() {
  return (
    <Stack.Navigator
      screenOptions={{
        headerShown: false,
        contentStyle: { backgroundColor: colors.background },
        ...stackGestureBackOptions,
      }}
    >
      <Stack.Screen name="Welcome" component={WelcomeScreen} />
      <Stack.Screen name="InputSelection" component={InputSelectionScreen} />
      <Stack.Screen name="Connection" component={ConnectionScreen} />
      <Stack.Screen name="SourceConfiguration" component={SourceConfigurationScreen} />
      <Stack.Screen
        name="WhatsAppPreferences"
        component={WhatsAppPreferencesScreen}
        options={({ navigation }) =>
          createBackHeaderOptions({
            title: 'WhatsApp Contacts',
            navigation,
          })}
      />
      <Stack.Screen
        name="TelegramPreferences"
        component={TelegramPreferencesScreen}
        options={({ navigation }) =>
          createBackHeaderOptions({
            title: 'Telegram Chats',
            navigation,
          })}
      />
      <Stack.Screen
        name="GmailPreferences"
        component={GmailPreferencesScreen}
        options={({ navigation }) =>
          createBackHeaderOptions({
            title: 'Gmail Senders',
            navigation,
          })}
      />
      <Stack.Screen
        name="GoogleCalendarPreferences"
        component={GoogleCalendarPreferencesScreen}
        options={({ navigation }) =>
          createBackHeaderOptions({
            title: 'Google Calendar',
            navigation,
          })}
      />
    </Stack.Navigator>
  );
}
