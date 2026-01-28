const { getDefaultConfig } = require('expo/metro-config');

const config = getDefaultConfig(__dirname);

// Disable Expo Router - this app uses traditional React Navigation
config.resolver.unstable_enablePackageExports = false;

module.exports = config;
