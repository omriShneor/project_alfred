import React from 'react';
import {
  TouchableOpacity,
  Text,
  StyleSheet,
  ActivityIndicator,
  ViewStyle,
  TextStyle,
} from 'react-native';
import { colors } from '../../theme/colors';

interface ButtonProps {
  title: string;
  onPress: () => void;
  variant?: 'primary' | 'secondary' | 'success' | 'danger' | 'outline';
  size?: 'small' | 'medium' | 'large';
  disabled?: boolean;
  loading?: boolean;
  style?: ViewStyle;
  textStyle?: TextStyle;
}

export function Button({
  title,
  onPress,
  variant = 'primary',
  size = 'medium',
  disabled = false,
  loading = false,
  style,
  textStyle,
}: ButtonProps) {
  const buttonColors: Record<string, { bg: string; text: string }> = {
    primary: { bg: colors.primary, text: '#ffffff' },
    secondary: { bg: colors.border, text: colors.text },
    success: { bg: colors.success, text: '#ffffff' },
    danger: { bg: colors.danger, text: '#ffffff' },
    outline: { bg: 'transparent', text: colors.primary },
  };

  const sizeStyles: Record<string, ViewStyle & { fontSize: number }> = {
    small: { paddingVertical: 6, paddingHorizontal: 12, fontSize: 12 },
    medium: { paddingVertical: 10, paddingHorizontal: 16, fontSize: 14 },
    large: { paddingVertical: 14, paddingHorizontal: 20, fontSize: 16 },
  };

  const isOutline = variant === 'outline';
  const { bg, text } = buttonColors[variant];
  const sizeStyle = sizeStyles[size];

  return (
    <TouchableOpacity
      style={[
        styles.button,
        {
          backgroundColor: bg,
          paddingVertical: sizeStyle.paddingVertical,
          paddingHorizontal: sizeStyle.paddingHorizontal,
          borderWidth: isOutline ? 1 : 0,
          borderColor: isOutline ? colors.primary : undefined,
          opacity: disabled ? 0.5 : 1,
        },
        style,
      ]}
      onPress={onPress}
      disabled={disabled || loading}
      activeOpacity={0.7}
    >
      {loading ? (
        <ActivityIndicator color={text} size="small" />
      ) : (
        <Text
          style={[
            styles.text,
            { color: text, fontSize: sizeStyle.fontSize },
            textStyle,
          ]}
        >
          {title}
        </Text>
      )}
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  button: {
    borderRadius: 6,
    alignItems: 'center',
    justifyContent: 'center',
    flexDirection: 'row',
  },
  text: {
    fontWeight: '600',
  },
});
