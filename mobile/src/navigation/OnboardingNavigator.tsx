import React from 'react';
import { Pressable, View } from 'react-native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { Ionicons } from '@expo/vector-icons';
import { WelcomeScreen } from '../screens/onboarding/WelcomeScreen';
import { InputSelectionScreen } from '../screens/onboarding/InputSelectionScreen';
import { ConnectionScreen } from '../screens/onboarding/ConnectionScreen';
import { SourceConfigurationScreen } from '../screens/onboarding/SourceConfigurationScreen';
import { WhatsAppPreferencesScreen, TelegramPreferencesScreen, GmailPreferencesScreen } from '../screens/smart-calendar';
import type { PreferenceStackParamList } from './PreferenceStackNavigator';
import { colors } from '../theme/colors';

export type OnboardingParamList = {
  Welcome: undefined;
  InputSelection: undefined;
  Connection: { whatsappEnabled: boolean; telegramEnabled: boolean; gmailEnabled: boolean };
  SourceConfiguration: { whatsappEnabled: boolean; telegramEnabled: boolean; gmailEnabled: boolean };
  // Add preference screens from shared stack
  WhatsAppPreferences: undefined;
  TelegramPreferences: undefined;
  GmailPreferences: undefined;
};

const Stack = createNativeStackNavigator<OnboardingParamList>();

export function OnboardingNavigator() {
  return (
    <Stack.Navigator
      screenOptions={{
        headerShown: false,
        contentStyle: { backgroundColor: colors.background },
      }}
    >
      <Stack.Screen name="Welcome" component={WelcomeScreen} />
      <Stack.Screen name="InputSelection" component={InputSelectionScreen} />
      <Stack.Screen name="Connection" component={ConnectionScreen} />
      <Stack.Screen name="SourceConfiguration" component={SourceConfigurationScreen} />
      <Stack.Screen
        name="WhatsAppPreferences"
        component={WhatsAppPreferencesScreen}
        options={({ navigation }) => ({
          headerShown: true,
          title: 'WhatsApp Contacts',
          headerStyle: { backgroundColor: colors.background },
          headerTintColor: colors.text,
          headerShadowVisible: false,
          headerBackVisible: false,
          headerLeft: () => (
            <View>
              <Pressable onPress={() => navigation.goBack()}>
                <Ionicons name="chevron-back" size={28} color={colors.text} />
              </Pressable>
            </View>
          ),
        })}
      />
      <Stack.Screen
        name="TelegramPreferences"
        component={TelegramPreferencesScreen}
        options={({ navigation }) => ({
          headerShown: true,
          title: 'Telegram Chats',
          headerStyle: { backgroundColor: colors.background },
          headerTintColor: colors.text,
          headerShadowVisible: false,
          headerBackVisible: false,
          headerLeft: () => (
            <View>
              <Pressable onPress={() => navigation.goBack()}>
                <Ionicons name="chevron-back" size={28} color={colors.text} />
              </Pressable>
            </View>
          ),
        })}
      />
      <Stack.Screen
        name="GmailPreferences"
        component={GmailPreferencesScreen}
        options={({ navigation }) => ({
          headerShown: true,
          title: 'Gmail Senders',
          headerStyle: { backgroundColor: colors.background },
          headerTintColor: colors.text,
          headerShadowVisible: false,
          headerBackVisible: false,
          headerLeft: () => (
            <View>
              <Pressable onPress={() => navigation.goBack()}>
                <Ionicons name="chevron-back" size={28} color={colors.text} />
              </Pressable>
            </View>
          ),
        })}
      />
    </Stack.Navigator>
  );
}
