import React from 'react';
import { View, Text, StyleSheet, ViewStyle } from 'react-native';
import { badgeColors, statusColors, colors } from '../../theme/colors';

interface BadgeProps {
  label: string;
  variant?: 'sender' | 'group' | 'create' | 'update' | 'delete' | 'status' | 'custom';
  status?: string;
  bgColor?: string;
  textColor?: string;
  style?: ViewStyle;
}

export function Badge({
  label,
  variant = 'custom',
  status,
  bgColor,
  textColor,
  style,
}: BadgeProps) {
  let bg = bgColor || colors.border;
  let text = textColor || colors.text;

  if (variant !== 'custom' && variant !== 'status' && badgeColors[variant]) {
    bg = badgeColors[variant].bg;
    text = badgeColors[variant].text;
  }

  if (variant === 'status' && status && statusColors[status]) {
    bg = statusColors[status] + '20'; // 20% opacity
    text = statusColors[status];
  }

  return (
    <View style={[styles.badge, { backgroundColor: bg }, style]}>
      <Text style={[styles.text, { color: text }]}>{label}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  badge: {
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderRadius: 4,
    alignSelf: 'flex-start',
  },
  text: {
    fontSize: 11,
    fontWeight: '600',
    textTransform: 'capitalize',
  },
});
