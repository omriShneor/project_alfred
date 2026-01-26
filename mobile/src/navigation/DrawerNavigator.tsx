import React from 'react';
import { createDrawerNavigator } from '@react-navigation/drawer';
import { HomeScreen } from '../screens/HomeScreen';
import { WhatsAppSettingsScreen } from '../screens/WhatsAppSettingsScreen';
import { GeneralSettingsScreen } from '../screens/GeneralSettingsScreen';
import { DrawerContent } from '../components/layout/DrawerContent';
import { colors } from '../theme/colors';

export type DrawerParamList = {
  Home: undefined;
  WhatsAppSettings: undefined;
  GeneralSettings: undefined;
};

const Drawer = createDrawerNavigator<DrawerParamList>();

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
        name="WhatsAppSettings"
        component={WhatsAppSettingsScreen}
        options={{ title: 'WhatsApp' }}
      />
      <Drawer.Screen
        name="GeneralSettings"
        component={GeneralSettingsScreen}
        options={{ title: 'General Settings' }}
      />
    </Drawer.Navigator>
  );
}
