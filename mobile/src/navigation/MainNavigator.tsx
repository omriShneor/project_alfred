import React from 'react';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { Ionicons } from '@expo/vector-icons';
import { HomeScreen } from '../screens/HomeScreen';
import { PreferencesScreen } from '../screens/PreferencesScreen';
import { SettingsScreen } from '../screens/SettingsScreen';
import { WhatsAppPreferencesScreen, GmailPreferencesScreen } from '../screens/smart-calendar';
import { colors } from '../theme/colors';

export type MainStackParamList = {
  Home: undefined;
  Tabs: undefined;
  WhatsAppPreferences: undefined;
  GmailPreferences: undefined;
};

export type TabParamList = {
  Preferences: undefined;
  Settings: undefined;
};

const Stack = createNativeStackNavigator<MainStackParamList>();
const Tab = createBottomTabNavigator<TabParamList>();

function TabNavigator() {
  return (
    <Tab.Navigator
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textSecondary,
        tabBarStyle: {
          backgroundColor: colors.card,
          borderTopColor: colors.border,
        },
      }}
    >
      <Tab.Screen
        name="Preferences"
        component={PreferencesScreen}
        options={{
          tabBarIcon: ({ color, size }) => (
            <Ionicons name="options-outline" size={size} color={color} />
          ),
        }}
      />
      <Tab.Screen
        name="Settings"
        component={SettingsScreen}
        options={{
          tabBarIcon: ({ color, size }) => (
            <Ionicons name="settings-outline" size={size} color={color} />
          ),
        }}
      />
    </Tab.Navigator>
  );
}

export function MainNavigator() {
  return (
    <Stack.Navigator
      initialRouteName="Tabs"
      screenOptions={{
        headerShown: false,
      }}
    >
      <Stack.Screen name="Home" component={HomeScreen} />
      <Stack.Screen name="Tabs" component={TabNavigator} />
      <Stack.Screen
        name="WhatsAppPreferences"
        component={WhatsAppPreferencesScreen}
        options={{
          headerShown: true,
          title: 'WhatsApp Sources',
          headerStyle: { backgroundColor: colors.background },
          headerTintColor: colors.text,
        }}
      />
      <Stack.Screen
        name="GmailPreferences"
        component={GmailPreferencesScreen}
        options={{
          headerShown: true,
          title: 'Gmail Sources',
          headerStyle: { backgroundColor: colors.background },
          headerTintColor: colors.text,
        }}
      />
    </Stack.Navigator>
  );
}
