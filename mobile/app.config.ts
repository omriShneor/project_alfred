import { ExpoConfig, ConfigContext } from 'expo/config';

export default ({ config }: ConfigContext): ExpoConfig => ({
  ...config,
  name: 'Project Alfred',
  slug: 'alfred-mobile',
  version: '1.0.0',
  orientation: 'portrait',
  icon: './assets/icon.png',
  userInterfaceStyle: 'light',
  newArchEnabled: true,
  splash: {
    image: './assets/splash-icon.png',
    resizeMode: 'contain',
    backgroundColor: '#3498db',
  },
  ios: {
    supportsTablet: true,
    bundleIdentifier: 'com.alfred.mobile',
    infoPlist: {
      CFBundleURLTypes: [
        {
          CFBundleURLSchemes: ['alfred'],
        },
      ],
    },
  },
  android: {
    adaptiveIcon: {
      foregroundImage: './assets/adaptive-icon.png',
      backgroundColor: '#3498db',
    },
    package: 'com.alfred.mobile',
    intentFilters: [
      {
        action: 'VIEW',
        autoVerify: true,
        data: [
          {
            scheme: 'alfred',
          },
        ],
        category: ['BROWSABLE', 'DEFAULT'],
      },
    ],
  },
  web: {
    favicon: './assets/favicon.png',
  },
  scheme: 'alfred',
  plugins: ['expo-web-browser'],
  extra: {
    apiBaseUrl: process.env.EXPO_PUBLIC_API_BASE_URL || 'http://localhost:8080',
    eas: {
      projectId: '99a272ff-7079-42d1-b71f-eb5078191f7e',
    },
  },
});
