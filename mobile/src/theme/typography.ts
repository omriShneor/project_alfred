import { StyleSheet } from 'react-native';
import { colors } from './colors';

export const typography = StyleSheet.create({
  h1: {
    fontSize: 24,
    fontWeight: '700',
    color: colors.text,
  },
  h2: {
    fontSize: 20,
    fontWeight: '600',
    color: colors.text,
  },
  h3: {
    fontSize: 16,
    fontWeight: '600',
    color: colors.text,
  },
  body: {
    fontSize: 14,
    color: colors.text,
  },
  bodySmall: {
    fontSize: 12,
    color: colors.textSecondary,
  },
  caption: {
    fontSize: 10,
    color: colors.textSecondary,
  },
});
