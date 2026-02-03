import {
  requestAdditionalScopes,
  exchangeAddScopesCode,
  type ScopeType,
} from '../../api/auth';
import { apiClient } from '../../api/client';

// Mock the API client
jest.mock('../../api/client', () => ({
  apiClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
  },
}));

const mockApiClient = apiClient as jest.Mocked<typeof apiClient>;

describe('Auth API - Incremental Authorization', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('requestAdditionalScopes', () => {
    it('requests Gmail scope without redirect URI', async () => {
      const mockResponse = { auth_url: 'https://accounts.google.com/oauth?scope=gmail.readonly' };
      mockApiClient.post.mockResolvedValueOnce(mockResponse);

      const result = await requestAdditionalScopes(['gmail']);

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/auth/google/add-scopes', {
        scopes: ['gmail'],
      });
      expect(result).toEqual(mockResponse);
    });

    it('requests Calendar scope without redirect URI', async () => {
      const mockResponse = { auth_url: 'https://accounts.google.com/oauth?scope=calendar' };
      mockApiClient.post.mockResolvedValueOnce(mockResponse);

      const result = await requestAdditionalScopes(['calendar']);

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/auth/google/add-scopes', {
        scopes: ['calendar'],
      });
      expect(result).toEqual(mockResponse);
    });

    it('requests Gmail scope with redirect URI', async () => {
      const mockResponse = { auth_url: 'https://accounts.google.com/oauth?scope=gmail.readonly' };
      mockApiClient.post.mockResolvedValueOnce(mockResponse);

      const result = await requestAdditionalScopes(['gmail'], 'https://example.com/callback');

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/auth/google/add-scopes', {
        scopes: ['gmail'],
        redirect_uri: 'https://example.com/callback',
      });
      expect(result).toEqual(mockResponse);
    });

    it('requests multiple scopes at once', async () => {
      const mockResponse = { auth_url: 'https://accounts.google.com/oauth?scope=gmail,calendar' };
      mockApiClient.post.mockResolvedValueOnce(mockResponse);

      const scopes: ScopeType[] = ['gmail', 'calendar'];
      const result = await requestAdditionalScopes(scopes);

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/auth/google/add-scopes', {
        scopes: ['gmail', 'calendar'],
      });
      expect(result).toEqual(mockResponse);
    });

    it('handles API error', async () => {
      const error = new Error('Network error');
      mockApiClient.post.mockRejectedValueOnce(error);

      await expect(requestAdditionalScopes(['gmail'])).rejects.toThrow('Network error');
    });
  });

  describe('exchangeAddScopesCode', () => {
    it('exchanges code for Gmail scope without redirect URI', async () => {
      const mockResponse = { status: 'scopes_added' };
      mockApiClient.post.mockResolvedValueOnce(mockResponse);

      const result = await exchangeAddScopesCode('test-auth-code', ['gmail']);

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/auth/google/add-scopes/callback', {
        code: 'test-auth-code',
        scopes: ['gmail'],
      });
      expect(result).toEqual(mockResponse);
    });

    it('exchanges code for Calendar scope without redirect URI', async () => {
      const mockResponse = { status: 'scopes_added' };
      mockApiClient.post.mockResolvedValueOnce(mockResponse);

      const result = await exchangeAddScopesCode('test-auth-code', ['calendar']);

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/auth/google/add-scopes/callback', {
        code: 'test-auth-code',
        scopes: ['calendar'],
      });
      expect(result).toEqual(mockResponse);
    });

    it('exchanges code with redirect URI', async () => {
      const mockResponse = { status: 'scopes_added' };
      mockApiClient.post.mockResolvedValueOnce(mockResponse);

      const result = await exchangeAddScopesCode(
        'test-auth-code',
        ['gmail'],
        'https://example.com/callback'
      );

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/auth/google/add-scopes/callback', {
        code: 'test-auth-code',
        scopes: ['gmail'],
        redirect_uri: 'https://example.com/callback',
      });
      expect(result).toEqual(mockResponse);
    });

    it('exchanges code for multiple scopes', async () => {
      const mockResponse = { status: 'scopes_added' };
      mockApiClient.post.mockResolvedValueOnce(mockResponse);

      const scopes: ScopeType[] = ['gmail', 'calendar'];
      const result = await exchangeAddScopesCode('test-auth-code', scopes);

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/auth/google/add-scopes/callback', {
        code: 'test-auth-code',
        scopes: ['gmail', 'calendar'],
      });
      expect(result).toEqual(mockResponse);
    });

    it('handles API error during code exchange', async () => {
      const error = new Error('Invalid authorization code');
      mockApiClient.post.mockRejectedValueOnce(error);

      await expect(exchangeAddScopesCode('invalid-code', ['gmail'])).rejects.toThrow(
        'Invalid authorization code'
      );
    });

    it('handles failed scope addition', async () => {
      const error = new Error('failed to add scopes: token exchange failed');
      mockApiClient.post.mockRejectedValueOnce(error);

      await expect(exchangeAddScopesCode('test-code', ['calendar'])).rejects.toThrow(
        'failed to add scopes'
      );
    });
  });

  describe('ScopeType', () => {
    it('accepts gmail as valid scope', async () => {
      const mockResponse = { auth_url: 'https://example.com' };
      mockApiClient.post.mockResolvedValueOnce(mockResponse);

      const scope: ScopeType = 'gmail';
      await requestAdditionalScopes([scope]);

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/auth/google/add-scopes', {
        scopes: ['gmail'],
      });
    });

    it('accepts calendar as valid scope', async () => {
      const mockResponse = { auth_url: 'https://example.com' };
      mockApiClient.post.mockResolvedValueOnce(mockResponse);

      const scope: ScopeType = 'calendar';
      await requestAdditionalScopes([scope]);

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/auth/google/add-scopes', {
        scopes: ['calendar'],
      });
    });
  });
});
