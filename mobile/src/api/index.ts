export { apiClient } from './client';
export { getHealth, type HealthStatus } from './health';
export { listChannels, createChannel, updateChannel, deleteChannel, discoverChannels } from './channels';
export { listEvents, getEvent, updateEvent, confirmEvent, rejectEvent, getChannelHistory, listCalendars, type ListEventsParams } from './events';
export { getWhatsAppStatus, generatePairingCode, disconnectWhatsApp, reconnectWhatsApp, type WhatsAppStatus, type PairingCodeResponse } from './whatsapp';
export { getGCalStatus, getOAuthURL, exchangeOAuthCode, disconnectGCal, type GCalStatus, type GCalConnectResponse } from './gcal';
export { getNotificationPrefs, updateEmailPrefs, registerPushToken, updatePushPrefs, type NotificationPreferences, type NotificationPrefsResponse } from './notifications';
export { getOnboardingStatus, type OnboardingStatus } from './onboarding';
export {
  getGmailStatus,
  getGmailSettings,
  updateGmailSettings,
  discoverCategories,
  discoverSenders,
  discoverDomains,
  listEmailSources,
  createEmailSource,
  getEmailSource,
  updateEmailSource,
  deleteEmailSource,
  type GmailStatus,
  type GmailSettings,
  type EmailSource,
  type EmailSourceType,
  type CreateEmailSourceRequest,
  type UpdateEmailSourceRequest,
  type DiscoveredCategory,
  type DiscoveredSender,
  type DiscoveredDomain,
} from './gmail';
export { getFeatures, updateSmartCalendar, getSmartCalendarStatus } from './features';
export { getAppStatus, completeOnboarding } from './app';
export {
  getTelegramStatus,
  sendTelegramCode,
  verifyTelegramCode,
  disconnectTelegram,
  reconnectTelegram,
  discoverTelegramChannels,
  listTelegramChannels,
  createTelegramChannel,
  updateTelegramChannel,
  deleteTelegramChannel,
  type TelegramStatus,
  type CreateTelegramChannelRequest,
  type UpdateTelegramChannelRequest,
} from './telegram';
export type { AppStatus, CompleteOnboardingRequest, ConnectionStatus } from '../types/app';
