import { useState, useEffect, useRef, useCallback } from 'react';
import { Platform } from 'react-native';
import * as Device from 'expo-device';
import * as Notifications from 'expo-notifications';
import Constants from 'expo-constants';
import { registerPushToken, updatePushPrefs } from '../api/notifications';
import { navigate } from '../navigation/navigationRef';

export interface PushNotificationState {
  expoPushToken: string | null;
  permissionStatus: 'granted' | 'denied' | 'undetermined';
  isRegistering: boolean;
  error: string | null;
}

export function usePushNotifications() {
  const [expoPushToken, setExpoPushToken] = useState<string | null>(null);
  const [permissionStatus, setPermissionStatus] = useState<'granted' | 'denied' | 'undetermined'>('undetermined');
  const [isRegistering, setIsRegistering] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const notificationListener = useRef<Notifications.EventSubscription | null>(null);
  const responseListener = useRef<Notifications.EventSubscription | null>(null);

  // Check initial permission status
  useEffect(() => {
    checkPermissionStatus();
  }, []);

  // Set up notification listeners
  useEffect(() => {
    // Listener for incoming notifications (foreground)
    notificationListener.current = Notifications.addNotificationReceivedListener(notification => {
      console.log('Notification received:', notification);
    });

    // Listener for notification interactions (tap)
    responseListener.current = Notifications.addNotificationResponseReceivedListener(response => {
      console.log('Notification response:', response);
      const data = response.notification.request.content.data;
      // Navigate based on screen specified in notification data
      if (data?.screen === 'Permissions') {
        console.log('Navigating to Connect Your Apps screen');
        navigate('SmartCalendarStack', { screen: 'Permissions' });
      } else if (data?.screen === 'SmartCalendar') {
        console.log('Navigating to Smart Calendar');
        navigate('SmartCalendarHub');
      } else if (data?.screen === 'Events') {
        console.log('Should navigate to Events screen, event ID:', data.eventId);
      }
    });

    return () => {
      if (notificationListener.current) {
        notificationListener.current.remove();
      }
      if (responseListener.current) {
        responseListener.current.remove();
      }
    };
  }, []);

  const checkPermissionStatus = async () => {
    if (Platform.OS === 'web') {
      setPermissionStatus('denied');
      setError('Push notifications not available on web');
      return;
    }

    const { status } = await Notifications.getPermissionsAsync();
    setPermissionStatus(status as 'granted' | 'denied' | 'undetermined');
  };

  const requestPermissions = useCallback(async (): Promise<boolean> => {
    if (Platform.OS === 'web') {
      setError('Push notifications not available on web');
      return false;
    }

    if (!Device.isDevice) {
      setError('Push notifications require a physical device');
      return false;
    }

    setIsRegistering(true);
    setError(null);

    try {
      // Check existing permissions
      const { status: existingStatus } = await Notifications.getPermissionsAsync();
      let finalStatus = existingStatus;

      // Request permissions if not granted
      if (existingStatus !== 'granted') {
        const { status } = await Notifications.requestPermissionsAsync();
        finalStatus = status;
      }

      setPermissionStatus(finalStatus as 'granted' | 'denied' | 'undetermined');

      if (finalStatus !== 'granted') {
        setIsRegistering(false);
        setError('Permission not granted for push notifications');
        return false;
      }

      // Get the Expo push token
      const projectId = Constants.expoConfig?.extra?.eas?.projectId;
      const tokenData = await Notifications.getExpoPushTokenAsync({
        projectId: projectId,
      });
      const token = tokenData.data;

      // Register token with backend
      await registerPushToken(token);

      // Enable push notifications
      await updatePushPrefs(true);

      setExpoPushToken(token);
      setIsRegistering(false);
      setError(null);

      return true;
    } catch (err: any) {
      console.error('Error requesting push permissions:', err);
      setIsRegistering(false);
      setError(err.message || 'Failed to setup push notifications');
      return false;
    }
  }, []);

  const setPushEnabled = useCallback(async (enabled: boolean): Promise<boolean> => {
    try {
      // If enabling and no token, request permissions first
      if (enabled && !expoPushToken) {
        return await requestPermissions();
      }

      // Update backend preference
      await updatePushPrefs(enabled);
      return true;
    } catch (err: any) {
      console.error('Error updating push preferences:', err);
      setError(err.message || 'Failed to update push preferences');
      return false;
    }
  }, [expoPushToken, requestPermissions]);

  return {
    expoPushToken,
    permissionStatus,
    isRegistering,
    error,
    requestPermissions,
    setPushEnabled,
    checkPermissionStatus,
  };
}
