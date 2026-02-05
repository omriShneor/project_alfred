import React from 'react';
import { Pressable, View } from 'react-native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { Ionicons } from '@expo/vector-icons';
import { HomeScreen } from '../screens/HomeScreen';
import { PreferencesScreen } from '../screens/PreferencesScreen';
import { SettingsScreen } from '../screens/SettingsScreen';
import { WhatsAppPreferencesScreen, TelegramPreferencesScreen, GmailPreferencesScreen, GoogleCalendarPreferencesScreen } from '../screens/smart-calendar';
import type { PreferenceStackParamList } from './PreferenceStackNavigator';
import { colors } from '../theme/colors';

export type TabParamList = {
  Home: undefined;
  Preferences: undefined;
  Settings: undefined;
};

export type MainStackParamList = {
  MainTabs: undefined;
  // Use shared types for preference screens
} & PreferenceStackParamList;

const Stack = createNativeStackNavigator<MainStackParamList>();
const Tab = createBottomTabNavigator<TabParamList>();

function TabNavigator() {
  return (
    <Tab.Navigator
      initialRouteName="Home"
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
          tabBarButton: () => null,
          tabBarItemStyle: { display: 'none' },
        }}
      />
      <Tab.Screen
        name="Preferences"
        component={PreferencesScreen}
        options={{
          tabBarLabel: 'Sources',
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
      initialRouteName="MainTabs"
      screenOptions={{
        headerShown: false,
      }}
    >
      <Stack.Screen name="MainTabs" component={TabNavigator} />
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
          title: 'Telegram Sources',
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
          title: 'Gmail Sources',
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
        name="GoogleCalendarPreferences"
        component={GoogleCalendarPreferencesScreen}
        options={({ navigation }) => ({
          headerShown: true,
          title: 'Google Calendar',
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
