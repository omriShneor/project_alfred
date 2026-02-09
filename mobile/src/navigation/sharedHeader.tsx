import React, { useCallback, useMemo } from 'react';
import { PanResponder, Pressable, StyleSheet } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import type { NativeStackNavigationOptions } from '@react-navigation/native-stack';
import { colors } from '../theme/colors';

const styles = StyleSheet.create({
  backButton: {
    alignItems: 'center',
    justifyContent: 'center',
    marginLeft: -4,
    minHeight: 36,
    minWidth: 36,
  },
});

export const backHeaderBaseOptions = {
  headerStyle: { backgroundColor: colors.background },
  headerTintColor: colors.text,
  headerShadowVisible: false,
  headerBackVisible: false,
  headerLeftContainerStyle: { paddingLeft: 8 },
};

export const stackGestureBackOptions: Pick<
  NativeStackNavigationOptions,
  'gestureEnabled' | 'fullScreenGestureEnabled'
> = {
  gestureEnabled: true,
  fullScreenGestureEnabled: true,
};

export function renderHeaderBackButton(onPress: () => void) {
  return () => (
    <Pressable
      onPress={onPress}
      style={styles.backButton}
      hitSlop={8}
      accessibilityRole="button"
      accessibilityLabel="Go back"
    >
      <Ionicons name="chevron-back" size={28} color={colors.text} />
    </Pressable>
  );
}

interface BackNavigation {
  canGoBack: () => boolean;
  goBack: () => void;
  getParent: () => BackNavigation | undefined;
  navigate: (...args: any[]) => void;
}

export function goBackWithFallback(
  navigation: BackNavigation,
  fallbackRouteName?: string
) {
  if (navigation.canGoBack()) {
    navigation.goBack();
    return;
  }

  const parentNavigation = navigation.getParent();
  if (parentNavigation?.canGoBack()) {
    parentNavigation.goBack();
    return;
  }

  if (fallbackRouteName) {
    const targetNavigation = parentNavigation ?? navigation;
    targetNavigation.navigate(fallbackRouteName);
  }
}

export function createBackHeaderOptions({
  title,
  navigation,
  fallbackRouteName,
}: {
  title: string;
  navigation: BackNavigation;
  fallbackRouteName?: string;
}) {
  return {
    headerShown: true as const,
    title,
    ...backHeaderBaseOptions,
    headerLeft: renderHeaderBackButton(() =>
      goBackWithFallback(navigation, fallbackRouteName)
    ),
  };
}

export function useEdgeSwipeBack(
  navigation: BackNavigation,
  {
    fallbackRouteName,
    edgeWidth = 24,
    activationDx = 12,
    completionDx = 72,
    maxVerticalDy = 120,
  }: {
    fallbackRouteName?: string;
    edgeWidth?: number;
    activationDx?: number;
    completionDx?: number;
    maxVerticalDy?: number;
  } = {}
) {
  const onBackPress = useCallback(
    () => goBackWithFallback(navigation, fallbackRouteName),
    [navigation, fallbackRouteName]
  );

  const panResponder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, gestureState) => {
          const startedFromLeftEdge = gestureState.x0 <= edgeWidth;
          const swipingRight = gestureState.dx > activationDx;
          const horizontalDominant =
            Math.abs(gestureState.dx) > Math.abs(gestureState.dy) * 1.5;

          return startedFromLeftEdge && swipingRight && horizontalDominant;
        },
        onPanResponderRelease: (_, gestureState) => {
          const completedSwipe =
            gestureState.dx > completionDx &&
            Math.abs(gestureState.dy) < maxVerticalDy;

          if (completedSwipe) {
            onBackPress();
          }
        },
      }),
    [activationDx, completionDx, edgeWidth, maxVerticalDy, onBackPress]
  );

  return {
    onBackPress,
    panHandlers: panResponder.panHandlers,
  };
}
