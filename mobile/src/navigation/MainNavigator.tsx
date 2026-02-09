import React from 'react';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { Ionicons } from '@expo/vector-icons';
import { HomeScreen } from '../screens/HomeScreen';
import { PreferencesScreen } from '../screens/PreferencesScreen';
import { SettingsScreen } from '../screens/SettingsScreen';
import { NeedsReviewScreen } from '../screens/NeedsReviewScreen';
import { WhatsAppPreferencesScreen, TelegramPreferencesScreen, GmailPreferencesScreen, GoogleCalendarPreferencesScreen } from '../screens/smart-calendar';
import type { PreferenceStackParamList } from './PreferenceStackNavigator';
import { colors } from '../theme/colors';
import {
  createBackHeaderOptions,
  stackGestureBackOptions,
} from './sharedHeader';

export type TabParamList = {
  Home: undefined;
  Preferences:
    | {
        reauthSources?: Array<
          'whatsapp' | 'telegram' | 'gmail' | 'google_calendar'
        >;
      }
    | undefined;
};

export type MainStackParamList = {
  MainTabs: undefined;
  Settings: undefined;
  NeedsReview: undefined;
  // Use shared types for preference screens
} & PreferenceStackParamList;

const Stack = createNativeStackNavigator<MainStackParamList>();
const Tab = createBottomTabNavigator<TabParamList>();

function TabNavigator() {
  return (
    <Tab.Navigator
      initialRouteName="Home"
      backBehavior="history"
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textSecondary,
        tabBarStyle: {
          backgroundColor: colors.card,
          borderTopColor: colors.border,
        },
        tabBarItemStyle: {
          flex: 1,
        },
      }}
    >
      <Tab.Screen
        name="Home"
        component={HomeScreen}
        options={{
          tabBarLabel: 'Home',
          tabBarIcon: ({ color, size }) => (
            <Ionicons name="home-outline" size={size} color={color} />
          ),
        }}
      />
      <Tab.Screen
        name="Preferences"
        component={PreferencesScreen}
        options={({ navigation }) => ({
          ...createBackHeaderOptions({
            title: 'Connections',
            navigation,
            fallbackRouteName: 'Home',
          }),
          tabBarLabel: 'Connections',
          tabBarIcon: ({ color, size }) => (
            <Ionicons name="options-outline" size={size} color={color} />
          ),
        })}
      />
    </Tab.Navigator>
  );
}

export function MainNavigator() {
  return (
    <Stack.Navigator
      initialRouteName="MainTabs"
      screenOptions={{
        headerShown: false,
        ...stackGestureBackOptions,
      }}
    >
      <Stack.Screen name="MainTabs" component={TabNavigator} />
      <Stack.Screen
        name="Settings"
        component={SettingsScreen}
        options={({ navigation }) =>
          createBackHeaderOptions({
            title: 'Notifications',
            navigation,
          })}
      />
      <Stack.Screen
        name="NeedsReview"
        component={NeedsReviewScreen}
        options={({ navigation }) =>
          createBackHeaderOptions({
            title: 'Needs Review',
            navigation,
          })}
      />
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
