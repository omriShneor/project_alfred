import { apiClient } from './client';

export type ScopeType = 'profile' | 'gmail' | 'calendar';

interface AddScopesRequest {
  scopes: ScopeType[];
  redirect_uri?: string;
}

interface AddScopesResponse {
  auth_url: string;
}

interface AddScopesCallbackRequest {
  code: string;
  redirect_uri?: string;
  scopes: ScopeType[];
}

interface AddScopesCallbackResponse {
  status: string;
}

/**
 * Request login with profile scopes
 * Returns an OAuth URL for initial authentication
 */
export async function requestLogin(redirectUri?: string): Promise<AddScopesResponse> {
  const body: any = {};
  if (redirectUri) {
    body.redirect_uri = redirectUri;
  }
  return apiClient.post<AddScopesResponse>('/api/auth/google/login', body);
}

/**
 * Request incremental authorization for additional scopes (Gmail or Calendar)
 * Returns an OAuth URL that the user should be redirected to
 */
export async function requestAdditionalScopes(
  scopes: ScopeType[],
  redirectUri?: string
): Promise<AddScopesResponse> {
  const body: AddScopesRequest = { scopes };
  if (redirectUri) {
    body.redirect_uri = redirectUri;
  }
  return apiClient.post<AddScopesResponse>('/api/auth/google/add-scopes', body);
}

/**
 * Exchange the authorization code after incremental authorization
 * This merges the new scopes with the user's existing scopes
 */
export async function exchangeAddScopesCode(
  code: string,
  scopes: ScopeType[],
  redirectUri?: string
): Promise<AddScopesCallbackResponse> {
  const body: AddScopesCallbackRequest = { code, scopes };
  if (redirectUri) {
    body.redirect_uri = redirectUri;
  }
  return apiClient.post<AddScopesCallbackResponse>('/api/auth/google/add-scopes/callback', body);
}
