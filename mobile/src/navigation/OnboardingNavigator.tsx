import React from 'react';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { WelcomeScreen } from '../screens/onboarding/WelcomeScreen';
import { InputSelectionScreen } from '../screens/onboarding/InputSelectionScreen';
import { ConnectionScreen } from '../screens/onboarding/ConnectionScreen';
import { colors } from '../theme/colors';

export type OnboardingParamList = {
  Welcome: undefined;
  InputSelection: undefined;
  Connection: { whatsappEnabled: boolean; gmailEnabled: boolean };
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
    </Stack.Navigator>
  );
}
