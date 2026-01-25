import React, { useState } from 'react';
import { View, TouchableOpacity, Text, StyleSheet, Linking } from 'react-native';
import { useHealth } from '../../hooks/useHealth';
import { Modal } from '../common/Modal';
import { Button } from '../common/Button';
import { colors } from '../../theme/colors';
import { WEB_SETTINGS_URL } from '../../config/api';

export function ConnectionStatus() {
  const { data: health, isError, isLoading } = useHealth();
  const [showModal, setShowModal] = useState(false);

  const whatsappConnected = health?.whatsapp === 'connected';
  const gcalConnected = health?.gcal === 'connected';
  const isConnected = whatsappConnected && gcalConnected && !isError;

  const handlePress = () => {
    if (!isConnected || isError) {
      setShowModal(true);
    }
  };

  const openWebSettings = () => {
    Linking.openURL(WEB_SETTINGS_URL);
    setShowModal(false);
  };

  if (isLoading) {
    return <View style={[styles.dot, styles.dotLoading]} />;
  }

  return (
    <>
      <TouchableOpacity onPress={handlePress} style={styles.container}>
        <View
          style={[
            styles.dot,
            { backgroundColor: isConnected ? colors.success : colors.danger },
          ]}
        />
      </TouchableOpacity>

      <Modal
        visible={showModal}
        onClose={() => setShowModal(false)}
        title="Connection Status"
      >
        <View style={styles.modalContent}>
          {isError ? (
            <View style={styles.statusRow}>
              <Text style={styles.statusLabel}>API</Text>
              <Text style={[styles.statusValue, { color: colors.danger }]}>
                Unreachable
              </Text>
            </View>
          ) : (
            <>
              <View style={styles.statusRow}>
                <Text style={styles.statusLabel}>WhatsApp</Text>
                <Text
                  style={[
                    styles.statusValue,
                    {
                      color: whatsappConnected
                        ? colors.success
                        : colors.danger,
                    },
                  ]}
                >
                  {whatsappConnected ? 'Connected' : 'Disconnected'}
                </Text>
              </View>

              <View style={styles.statusRow}>
                <Text style={styles.statusLabel}>Google Calendar</Text>
                <Text
                  style={[
                    styles.statusValue,
                    { color: gcalConnected ? colors.success : colors.danger },
                  ]}
                >
                  {gcalConnected ? 'Connected' : 'Disconnected'}
                </Text>
              </View>
            </>
          )}

          {(!isConnected || isError) && (
            <View style={styles.helpSection}>
              <Text style={styles.helpText}>
                {isError
                  ? 'Unable to reach the Alfred server. Please check your connection.'
                  : 'To reconnect, please use the web interface.'}
              </Text>
              <Button
                title="Open Web Settings"
                onPress={openWebSettings}
                variant="primary"
                style={styles.webButton}
              />
            </View>
          )}
        </View>
      </Modal>
    </>
  );
}

const styles = StyleSheet.create({
  container: {
    padding: 8,
  },
  dot: {
    width: 12,
    height: 12,
    borderRadius: 6,
  },
  dotLoading: {
    backgroundColor: colors.textSecondary,
  },
  modalContent: {
    paddingBottom: 20,
  },
  statusRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: 12,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
  },
  statusLabel: {
    fontSize: 15,
    color: colors.text,
  },
  statusValue: {
    fontSize: 15,
    fontWeight: '600',
  },
  helpSection: {
    marginTop: 20,
  },
  helpText: {
    fontSize: 14,
    color: colors.textSecondary,
    textAlign: 'center',
    marginBottom: 16,
  },
  webButton: {
    marginTop: 8,
  },
});
