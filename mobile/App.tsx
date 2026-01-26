import React from 'react';
import { StatusBar } from 'expo-status-bar';
import { StyleSheet, View } from 'react-native';
import { NavigationContainer } from '@react-navigation/native';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import * as Notifications from 'expo-notifications';

import { Header } from './src/components/layout/Header';
import { RootNavigator } from './src/navigation/RootNavigator';
import { AppStateProvider, useAppState } from './src/context/AppStateContext';
import { colors } from './src/theme/colors';

// Configure notification handler for foreground notifications
Notifications.setNotificationHandler({
  handleNotification: async () => ({
    shouldShowAlert: true,
    shouldPlaySound: true,
    shouldSetBadge: true,
    shouldShowBanner: true,
    shouldShowList: true,
  }),
});

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 2,
      staleTime: 30000,
      refetchOnWindowFocus: true,
    },
    mutations: {
      retry: 1,
    },
  },
});

function AppContent() {
  const { showDrawerToggle } = useAppState();

  return (
    <View style={styles.container}>
      <StatusBar style="dark" />
      <Header showDrawerToggle={showDrawerToggle} />
      <RootNavigator />
    </View>
  );
}

export default function App() {
  return (
    <GestureHandlerRootView style={styles.gestureRoot}>
      <QueryClientProvider client={queryClient}>
        <SafeAreaProvider>
          <AppStateProvider>
            <NavigationContainer>
              <AppContent />
            </NavigationContainer>
          </AppStateProvider>
        </SafeAreaProvider>
      </QueryClientProvider>
    </GestureHandlerRootView>
  );
}

const styles = StyleSheet.create({
  gestureRoot: {
    flex: 1,
  },
  container: {
    flex: 1,
    backgroundColor: colors.background,
  },
});
