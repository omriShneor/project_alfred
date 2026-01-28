// Integration status for an input or calendar
export interface IntegrationStatus {
  enabled: boolean;
  status: 'pending' | 'connecting' | 'available' | 'error';
}

// Smart Calendar inputs (where to scan for events)
export interface SmartCalendarInputs {
  whatsapp: IntegrationStatus;
  email: IntegrationStatus;
  sms: IntegrationStatus;
}

// Smart Calendar calendars (where to sync events)
export interface SmartCalendarCalendars {
  alfred: IntegrationStatus;         // Local Alfred calendar (always available)
  google_calendar: IntegrationStatus;
  outlook: IntegrationStatus;
}

// Smart Calendar feature settings
export interface SmartCalendarFeature {
  enabled: boolean;
  setup_complete: boolean;
  inputs: SmartCalendarInputs;
  calendars: SmartCalendarCalendars;
}

// All feature settings
export interface FeaturesResponse {
  smart_calendar: SmartCalendarFeature;
}

// Request to update Smart Calendar settings
export interface UpdateSmartCalendarRequest {
  enabled?: boolean;
  setup_complete?: boolean;
  inputs?: {
    whatsapp: boolean;
    email: boolean;
    sms: boolean;
  };
  calendars?: {
    alfred: boolean;
    google_calendar: boolean;
    outlook: boolean;
  };
}

// Detailed status of Smart Calendar integrations
export interface SmartCalendarStatusIntegration {
  type: 'input' | 'calendar';
  name: string;
  status: 'pending' | 'connecting' | 'available' | 'error';
  message: string;
}

export interface SmartCalendarStatusResponse {
  enabled: boolean;
  setup_complete: boolean;
  integrations: SmartCalendarStatusIntegration[];
  all_available: boolean;
}
