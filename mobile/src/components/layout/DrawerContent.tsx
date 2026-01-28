import React from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
import { DrawerContentScrollView } from '@react-navigation/drawer';
import { Feather } from '@expo/vector-icons';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { colors } from '../../theme/colors';
import { useSmartCalendarEnabled } from '../../hooks';
import type { DrawerContentComponentProps } from '@react-navigation/drawer';

interface DrawerItemProps {
  label: string;
  icon: keyof typeof Feather.glyphMap;
  onPress: () => void;
  isActive?: boolean;
}

function DrawerItem({ label, icon, onPress, isActive }: DrawerItemProps) {
  return (
    <TouchableOpacity
      style={[
        styles.drawerItem,
        isActive && styles.drawerItemActive,
      ]}
      onPress={onPress}
      activeOpacity={0.7}
    >
      <Feather
        name={icon}
        size={20}
        color={isActive ? colors.primary : colors.textSecondary}
      />
      <Text
        style={[
          styles.drawerItemLabel,
          isActive && styles.drawerItemLabelActive,
        ]}
      >
        {label}
      </Text>
    </TouchableOpacity>
  );
}

export function DrawerContent(props: DrawerContentComponentProps) {
  const insets = useSafeAreaInsets();
  const { state, navigation } = props;
  const { isReady: smartCalendarReady } = useSmartCalendarEnabled();

  const currentRoute = state.routes[state.index].name;

  const navigateToScreen = (screenName: string) => {
    navigation.navigate(screenName);
  };

  return (
    <DrawerContentScrollView
      {...props}
      contentContainerStyle={[styles.container, { paddingTop: insets.top + 16 }]}
    >
      <View style={styles.menu}>
        {/* Home - always first */}
        <DrawerItem
          label="Home"
          icon="home"
          onPress={() => navigateToScreen('Home')}
          isActive={currentRoute === 'Home'}
        />

        {/* Assistant Capabilities - always second */}
        <DrawerItem
          label="Assistant Capabilities"
          icon="sliders"
          onPress={() => navigateToScreen('Capabilities')}
          isActive={currentRoute === 'Capabilities'}
        />

        {/* Smart Calendar - only if enabled and setup complete */}
        {smartCalendarReady && (
          <DrawerItem
            label="Smart Calendar"
            icon="calendar"
            onPress={() => navigateToScreen('SmartCalendarHub')}
            isActive={currentRoute === 'SmartCalendarHub'}
          />
        )}
      </View>
    </DrawerContentScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  menu: {
    paddingHorizontal: 12,
  },
  drawerItem: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 12,
    paddingHorizontal: 12,
    borderRadius: 8,
    marginBottom: 4,
  },
  drawerItemActive: {
    backgroundColor: colors.primary + '15',
  },
  drawerItemLabel: {
    fontSize: 15,
    color: colors.textSecondary,
    marginLeft: 12,
    fontWeight: '500',
  },
  drawerItemLabelActive: {
    color: colors.primary,
    fontWeight: '600',
  },
});
