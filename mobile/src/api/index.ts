export { apiClient } from './client';
export { getHealth, type HealthStatus } from './health';
export { requestAdditionalScopes, exchangeAddScopesCode, type ScopeType } from './auth';
export { listChannels, createChannel, updateChannel, deleteChannel } from './channels';
export { listEvents, getEvent, updateEvent, confirmEvent, rejectEvent, getChannelHistory, listCalendars, type ListEventsParams } from './events';
export { getWhatsAppStatus, generatePairingCode, disconnectWhatsApp, reconnectWhatsApp, type WhatsAppStatus, type PairingCodeResponse } from './whatsapp';
export { getGCalStatus, getOAuthURL, exchangeOAuthCode, disconnectGScope, getGCalSettings, updateGCalSettings, type GCalStatus, type GCalConnectResponse, type GCalSettings, type UpdateGCalSettingsRequest } from './gcal';
export { getNotificationPrefs, updateEmailPrefs, registerPushToken, updatePushPrefs, type NotificationPreferences, type NotificationPrefsResponse } from './notifications';
export { getOnboardingStatus, type OnboardingStatus } from './onboarding';
export {
  getGmailStatus,
  listEmailSources,
  createEmailSource,
  updateEmailSource,
  deleteEmailSource,
  getTopContacts,
  addCustomSource,
  type GmailStatus,
  type EmailSource,
  type EmailSourceType,
  type CreateEmailSourceRequest,
  type UpdateEmailSourceRequest,
  type TopContact,
  type AddCustomSourceRequest,
} from './gmail';
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
