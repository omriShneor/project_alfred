import React from 'react';
import { createMaterialTopTabNavigator } from '@react-navigation/material-top-tabs';
import { ChannelsScreen } from '../screens/ChannelsScreen';
import { EventsScreen } from '../screens/EventsScreen';
import { colors } from '../theme/colors';

const Tab = createMaterialTopTabNavigator();

export function TopTabs() {
  return (
    <Tab.Navigator
      screenOptions={{
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textSecondary,
        tabBarIndicatorStyle: {
          backgroundColor: colors.primary,
          height: 3,
        },
        tabBarStyle: {
          backgroundColor: colors.card,
          elevation: 0,
          shadowOpacity: 0,
          borderBottomWidth: 1,
          borderBottomColor: colors.border,
        },
        tabBarLabelStyle: {
          fontSize: 14,
          fontWeight: '600',
          textTransform: 'none',
        },
        tabBarPressColor: colors.primary + '20',
      }}
    >
      <Tab.Screen
        name="Channels"
        component={ChannelsScreen}
        options={{ tabBarLabel: 'Channels' }}
      />
      <Tab.Screen
        name="Events"
        component={EventsScreen}
        options={{ tabBarLabel: 'Events' }}
      />
    </Tab.Navigator>
  );
}
