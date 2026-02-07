import React from 'react';
import {
  Modal as RNModal,
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Feather } from '@expo/vector-icons';
import { colors } from '../../theme/colors';

interface ModalProps {
  visible: boolean;
  onClose: () => void;
  title?: string;
  children: React.ReactNode;
  scrollable?: boolean;
  footer?: React.ReactNode;
}

export function Modal({ visible, onClose, title, children, scrollable = true, footer }: ModalProps) {
  const insets = useSafeAreaInsets();
  const ContentWrapper = scrollable ? ScrollView : View;
  const contentProps = scrollable
    ? { keyboardShouldPersistTaps: 'handled' as const, contentContainerStyle: styles.scrollContent }
    : {};

  return (
    <RNModal
      visible={visible}
      animationType="slide"
      onRequestClose={onClose}
    >
      <View style={[styles.safeArea, { paddingTop: insets.top }]}>
        <KeyboardAvoidingView
          behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
          style={styles.container}
        >
          <View style={styles.header}>
            <TouchableOpacity onPress={onClose} style={styles.backButton} hitSlop={{ top: 8, bottom: 8, left: 8, right: 8 }}>
              <Feather name="arrow-left" size={24} color={colors.text} />
            </TouchableOpacity>
            {title && <Text style={styles.title}>{title}</Text>}
            <View style={styles.headerSpacer} />
          </View>
          <ContentWrapper style={styles.content} {...contentProps}>
            {children}
          </ContentWrapper>
          {footer}
        </KeyboardAvoidingView>
      </View>
    </RNModal>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: colors.background,
  },
  container: {
    flex: 1,
    backgroundColor: colors.background,
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
    backgroundColor: colors.background,
  },
  backButton: {
    padding: 4,
    marginRight: 12,
  },
  title: {
    flex: 1,
    fontSize: 18,
    fontWeight: '600',
    color: colors.text,
  },
  headerSpacer: {
    width: 32,
  },
  content: {
    flex: 1,
  },
  scrollContent: {
    padding: 16,
    flexGrow: 1,
  },
});
