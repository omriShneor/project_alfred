import React from 'react';
import { createDrawerNavigator } from '@react-navigation/drawer';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { HomeScreen } from '../screens/HomeScreen';
import { CapabilitiesScreen } from '../screens/CapabilitiesScreen';
import {
  SmartCalendarScreen,
  SmartCalendarSetupScreen,
  SmartCalendarPermissionsScreen,
} from '../screens/smart-calendar';
import { DrawerContent } from '../components/layout/DrawerContent';
import { colors } from '../theme/colors';

export type DrawerParamList = {
  Home: undefined;
  Capabilities: undefined;
  SmartCalendar: undefined;
  SmartCalendarStack: undefined;
};

export type SmartCalendarStackParamList = {
  Setup: undefined;
  Permissions: undefined;
  SmartCalendarMain: undefined;
};

const Drawer = createDrawerNavigator<DrawerParamList>();
const SmartCalendarStack = createNativeStackNavigator<SmartCalendarStackParamList>();

// Stack navigator for Smart Calendar setup flow
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
      <SmartCalendarStack.Screen
        name="SmartCalendarMain"
        component={SmartCalendarScreen}
        options={{ title: 'Smart Calendar', headerShown: false }}
      />
    </SmartCalendarStack.Navigator>
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
        name="SmartCalendar"
        component={SmartCalendarScreen}
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
