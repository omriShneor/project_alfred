// Skip auto-detection of host component names (incompatible with react-native-web)
// Must be set before importing @testing-library/react-native
process.env.RNTL_SKIP_AUTO_DETECT_HOST_COMPONENT_NAMES = 'true';

// Configure @testing-library/react-native with web component names
const { configure } = require('@testing-library/react-native');

// Configure with web host component names
// react-native-web renders Text as nested divs, not spans
configure({
  asyncUtilTimeout: 5000,
  hostComponentNames: {
    text: 'div',
    textInput: 'input',
    switch: 'input',
    scrollView: 'div',
    modal: 'div',
    image: 'img',
  },
});

// Mock expo-constants
jest.mock('expo-constants', () => ({
  expoConfig: {
    extra: {
      apiBaseUrl: 'http://localhost:8080',
    },
  },
}));

// Mock @expo/vector-icons (requires expo-font which doesn't work in jsdom)
jest.mock('@expo/vector-icons', () => {
  const React = require('react');
  const mockIcon = ({ name, size, color, ...props }) =>
    React.createElement('span', {
      'data-testid': `icon-${name}`,
      'data-icon-name': name,
      ...props
    });
  return {
    Feather: mockIcon,
    Ionicons: mockIcon,
    MaterialIcons: mockIcon,
    MaterialCommunityIcons: mockIcon,
    FontAwesome: mockIcon,
    AntDesign: mockIcon,
    Entypo: mockIcon,
    createIconSet: () => mockIcon,
  };
});

// Mock expo-clipboard
jest.mock('expo-clipboard', () => ({
  setStringAsync: jest.fn(),
  getStringAsync: jest.fn(),
}));

// Mock expo-linking
jest.mock('expo-linking', () => ({
  createURL: jest.fn((path) => `exp://localhost:8081/${path}`),
  openURL: jest.fn(),
}));

// Mock expo-web-browser
jest.mock('expo-web-browser', () => ({
  openAuthSessionAsync: jest.fn(),
  openBrowserAsync: jest.fn(),
}));

// Mock expo-notifications
jest.mock('expo-notifications', () => ({
  getPermissionsAsync: jest.fn().mockResolvedValue({ status: 'granted' }),
  requestPermissionsAsync: jest.fn().mockResolvedValue({ status: 'granted' }),
  getExpoPushTokenAsync: jest.fn().mockResolvedValue({ data: 'test-push-token' }),
  setNotificationHandler: jest.fn(),
  addNotificationReceivedListener: jest.fn(() => ({ remove: jest.fn() })),
  addNotificationResponseReceivedListener: jest.fn(() => ({ remove: jest.fn() })),
}));

// Mock expo-device
jest.mock('expo-device', () => ({
  isDevice: true,
}));

// Mock @react-navigation/native
jest.mock('@react-navigation/native', () => {
  return {
    useNavigation: () => ({
      navigate: jest.fn(),
      goBack: jest.fn(),
      setOptions: jest.fn(),
    }),
    useRoute: () => ({
      params: {},
    }),
    useFocusEffect: jest.fn(),
    NavigationContainer: ({ children }) => children,
  };
});

// Mock @react-navigation/native-stack
jest.mock('@react-navigation/native-stack', () => ({
  createNativeStackNavigator: jest.fn(() => ({
    Navigator: ({ children }) => children,
    Screen: ({ children }) => children,
  })),
}));

// Mock @react-navigation/bottom-tabs
jest.mock('@react-navigation/bottom-tabs', () => ({
  createBottomTabNavigator: jest.fn(() => ({
    Navigator: ({ children }) => children,
    Screen: ({ children }) => children,
  })),
}));

// Mock safe area context
jest.mock('react-native-safe-area-context', () => {
  const inset = { top: 0, right: 0, bottom: 0, left: 0 };
  return {
    SafeAreaProvider: ({ children }) => children,
    SafeAreaView: ({ children }) => children,
    useSafeAreaInsets: () => inset,
  };
});

// Mock react-native-modal-datetime-picker
jest.mock('react-native-modal-datetime-picker', () => {
  const React = require('react');
  return {
    __esModule: true,
    default: ({ isVisible, onConfirm, onCancel }) => {
      if (!isVisible) return null;
      return React.createElement('View', { testID: 'datetime-picker' });
    },
  };
});

// Mock react-native-gesture-handler
jest.mock('react-native-gesture-handler', () => ({
  Swipeable: 'Swipeable',
  DrawerLayout: 'DrawerLayout',
  State: {},
  ScrollView: 'ScrollView',
  Slider: 'Slider',
  Switch: 'Switch',
  TextInput: 'TextInput',
  ToolbarAndroid: 'ToolbarAndroid',
  ViewPagerAndroid: 'ViewPagerAndroid',
  DrawerLayoutAndroid: 'DrawerLayoutAndroid',
  WebView: 'WebView',
  NativeViewGestureHandler: 'NativeViewGestureHandler',
  TapGestureHandler: 'TapGestureHandler',
  FlingGestureHandler: 'FlingGestureHandler',
  ForceTouchGestureHandler: 'ForceTouchGestureHandler',
  LongPressGestureHandler: 'LongPressGestureHandler',
  PanGestureHandler: 'PanGestureHandler',
  PinchGestureHandler: 'PinchGestureHandler',
  RotationGestureHandler: 'RotationGestureHandler',
  RawButton: 'RawButton',
  BaseButton: 'BaseButton',
  RectButton: 'RectButton',
  BorderlessButton: 'BorderlessButton',
  FlatList: 'FlatList',
  gestureHandlerRootHOC: jest.fn((component) => component),
  Directions: {},
}));

// Mock react-native-reanimated with a simple mock
jest.mock('react-native-reanimated', () => ({
  default: {
    call: jest.fn(),
    createAnimatedComponent: (component) => component,
    View: 'Animated.View',
    Text: 'Animated.Text',
    ScrollView: 'Animated.ScrollView',
  },
  useSharedValue: jest.fn(() => ({ value: 0 })),
  useAnimatedStyle: jest.fn(() => ({})),
  withTiming: jest.fn((val) => val),
  withSpring: jest.fn((val) => val),
  withDecay: jest.fn((val) => val),
  runOnJS: jest.fn((fn) => fn),
  runOnUI: jest.fn((fn) => fn),
  useAnimatedGestureHandler: jest.fn(() => ({})),
  useAnimatedScrollHandler: jest.fn(() => ({})),
  interpolate: jest.fn(),
  Extrapolate: {
    CLAMP: 'clamp',
    EXTEND: 'extend',
    IDENTITY: 'identity',
  },
}));

// Mock react-native-screens
jest.mock('react-native-screens', () => ({
  enableScreens: jest.fn(),
  Screen: 'Screen',
  ScreenContainer: 'ScreenContainer',
}));

// Mock Modal from react-native-web (Modal portals don't work in jsdom)
// We need to mock it within the react-native-web exports
jest.mock('react-native-web', () => {
  const actualRNW = jest.requireActual('react-native-web');
  const React = require('react');

  // Create a simple Modal mock that renders children when visible
  const MockModal = ({ visible, children, onRequestClose, transparent, animationType }) => {
    if (!visible) return null;
    return React.createElement('div', {
      'data-testid': 'modal',
      style: { position: 'fixed', top: 0, left: 0, right: 0, bottom: 0 }
    }, children);
  };

  // Create a mock Alert that stores calls for testing
  const MockAlert = {
    alert: jest.fn((title, message, buttons) => {
      // Auto-confirm by calling the last button (typically the confirm action)
      // This allows tests to verify the behavior
    }),
  };

  return {
    ...actualRNW,
    Modal: MockModal,
    Alert: MockAlert,
  };
});


// Mock fetch globally
global.fetch = jest.fn();

// Reset mocks before each test
beforeEach(() => {
  jest.clearAllMocks();
  global.fetch.mockReset();
});
