import React, { useState } from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
import { DrawerContentScrollView } from '@react-navigation/drawer';
import { Feather } from '@expo/vector-icons';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { colors } from '../../theme/colors';
import type { DrawerContentComponentProps } from '@react-navigation/drawer';

interface DrawerItemProps {
  label: string;
  icon: keyof typeof Feather.glyphMap;
  onPress: () => void;
  isActive?: boolean;
  indent?: boolean;
}

function DrawerItem({ label, icon, onPress, isActive, indent }: DrawerItemProps) {
  return (
    <TouchableOpacity
      style={[
        styles.drawerItem,
        isActive && styles.drawerItemActive,
        indent && styles.drawerItemIndented,
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
  const [settingsExpanded, setSettingsExpanded] = useState(true);
  const insets = useSafeAreaInsets();
  const { state, navigation } = props;

  const currentRoute = state.routes[state.index].name;

  const navigateToScreen = (screenName: string) => {
    navigation.navigate(screenName);
  };

  return (
    <DrawerContentScrollView
      {...props}
      contentContainerStyle={[styles.container, { paddingTop: insets.top + 16 }]}
    >
      <View style={styles.header}>
        <Text style={styles.headerTitle}>Project Alfred</Text>
      </View>

      <View style={styles.menu}>
        <DrawerItem
          label="Home"
          icon="home"
          onPress={() => navigateToScreen('Home')}
          isActive={currentRoute === 'Home'}
        />

        {/* Settings Section with expandable submenu */}
        <TouchableOpacity
          style={styles.expandableHeader}
          onPress={() => setSettingsExpanded(!settingsExpanded)}
          activeOpacity={0.7}
        >
          <View style={styles.expandableHeaderContent}>
            <Feather name="settings" size={20} color={colors.textSecondary} />
            <Text style={styles.expandableHeaderLabel}>Settings</Text>
          </View>
          <Feather
            name={settingsExpanded ? 'chevron-down' : 'chevron-right'}
            size={20}
            color={colors.textSecondary}
          />
        </TouchableOpacity>

        {settingsExpanded && (
          <View style={styles.submenu}>
            <DrawerItem
              label="WhatsApp"
              icon="message-circle"
              onPress={() => navigateToScreen('WhatsAppSettings')}
              isActive={currentRoute === 'WhatsAppSettings'}
              indent
            />
            <DrawerItem
              label="General"
              icon="sliders"
              onPress={() => navigateToScreen('GeneralSettings')}
              isActive={currentRoute === 'GeneralSettings'}
              indent
            />
          </View>
        )}
      </View>
    </DrawerContentScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  header: {
    paddingHorizontal: 20,
    paddingBottom: 20,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
    marginBottom: 16,
  },
  headerTitle: {
    fontSize: 20,
    fontWeight: '700',
    color: colors.text,
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
  drawerItemIndented: {
    marginLeft: 24,
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
  expandableHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 12,
    paddingHorizontal: 12,
    borderRadius: 8,
    marginBottom: 4,
  },
  expandableHeaderContent: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  expandableHeaderLabel: {
    fontSize: 15,
    color: colors.textSecondary,
    marginLeft: 12,
    fontWeight: '500',
  },
  submenu: {
    marginBottom: 4,
  },
});
