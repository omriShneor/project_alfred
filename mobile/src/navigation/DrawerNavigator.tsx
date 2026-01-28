import React from 'react';
import { createDrawerNavigator } from '@react-navigation/drawer';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { HomeScreen } from '../screens/HomeScreen';
import { CapabilitiesScreen } from '../screens/CapabilitiesScreen';
import {
  SmartCalendarSetupScreen,
  SmartCalendarPermissionsScreen,
  InputIntegrationsScreen,
  WhatsAppPreferencesScreen,
  GmailPreferencesScreen,
} from '../screens/smart-calendar';
import { DrawerContent } from '../components/layout/DrawerContent';
import { colors } from '../theme/colors';

export type DrawerParamList = {
  Home: undefined;
  Capabilities: undefined;
  SmartCalendarHub: undefined;
  SmartCalendarStack: undefined;
};

export type SmartCalendarStackParamList = {
  Setup: undefined;
  Permissions: undefined;
};

export type SmartCalendarHubStackParamList = {
  InputIntegrations: undefined;
  WhatsAppPreferences: undefined;
  GmailPreferences: undefined;
};

const Drawer = createDrawerNavigator<DrawerParamList>();
const SmartCalendarStack = createNativeStackNavigator<SmartCalendarStackParamList>();
const SmartCalendarHubStack = createNativeStackNavigator<SmartCalendarHubStackParamList>();

// Stack navigator for Smart Calendar setup flow (onboarding)
function SmartCalendarStackNavigator() {
  return (
    <SmartCalendarStack.Navigator
      screenOptions={{
        headerStyle: { backgroundColor: colors.background },
        headerShadowVisible: false,
        headerTintColor: colors.text,
      }}
    >
      <SmartCalendarStack.Screen
        name="Setup"
        component={SmartCalendarSetupScreen}
        options={{ title: 'Smart Calendar Setup' }}
      />
      <SmartCalendarStack.Screen
        name="Permissions"
        component={SmartCalendarPermissionsScreen}
        options={{ title: 'Connect Your Accounts' }}
      />
    </SmartCalendarStack.Navigator>
  );
}

// Stack navigator for Smart Calendar hub (input integrations)
function SmartCalendarHubNavigator() {
  return (
    <SmartCalendarHubStack.Navigator
      screenOptions={{
        headerStyle: { backgroundColor: colors.background },
        headerShadowVisible: false,
        headerTintColor: colors.text,
      }}
    >
      <SmartCalendarHubStack.Screen
        name="InputIntegrations"
        component={InputIntegrationsScreen}
        options={{ title: 'Smart Calendar' }}
      />
      <SmartCalendarHubStack.Screen
        name="WhatsAppPreferences"
        component={WhatsAppPreferencesScreen}
        options={{ title: 'WhatsApp Preferences' }}
      />
      <SmartCalendarHubStack.Screen
        name="GmailPreferences"
        component={GmailPreferencesScreen}
        options={{ title: 'Gmail Preferences' }}
      />
    </SmartCalendarHubStack.Navigator>
  );
}

export function DrawerNavigator() {
  return (
    <Drawer.Navigator
      initialRouteName="Home"
      drawerContent={(props) => <DrawerContent {...props} />}
      screenOptions={{
        headerShown: false,
        drawerStyle: {
          backgroundColor: colors.card,
          width: 280,
        },
        drawerType: 'front',
        overlayColor: 'rgba(0,0,0,0.5)',
      }}
    >
      <Drawer.Screen
        name="Home"
        component={HomeScreen}
        options={{ title: 'Home' }}
      />
      <Drawer.Screen
        name="Capabilities"
        component={CapabilitiesScreen}
        options={{ title: 'Assistant Capabilities' }}
      />
      <Drawer.Screen
        name="SmartCalendarHub"
        component={SmartCalendarHubNavigator}
        options={{ title: 'Smart Calendar' }}
      />
      <Drawer.Screen
        name="SmartCalendarStack"
        component={SmartCalendarStackNavigator}
        options={{
          title: 'Smart Calendar Setup',
          drawerItemStyle: { display: 'none' }, // Hidden from drawer
        }}
      />
    </Drawer.Navigator>
  );
}
