export interface ConnectionStatus {
  enabled: boolean;
  connected: boolean;
}

export interface AppStatus {
  onboarding_complete: boolean;
  whatsapp?: ConnectionStatus;
  telegram?: ConnectionStatus;
  gmail?: ConnectionStatus;
  google_calendar?: ConnectionStatus;
}

export interface CompleteOnboardingRequest {
  whatsapp_enabled: boolean;
  telegram_enabled: boolean;
  gmail_enabled: boolean;
}
