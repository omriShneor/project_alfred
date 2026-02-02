import { apiClient } from '../../api/client';

// Mock the config
jest.mock('../../config/api', () => ({
  API_BASE_URL: 'http://localhost:8080',
}));

describe('apiClient', () => {
  const mockFetch = global.fetch as jest.Mock;

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('get', () => {
    it('makes a GET request to the correct URL', async () => {
      const mockResponse = { data: 'test' };
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(mockResponse),
      });

      const result = await apiClient.get('/api/test');

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/test',
        expect.objectContaining({
          method: 'GET',
          headers: { 'Content-Type': 'application/json' },
          body: undefined,
        })
      );
      expect(result).toEqual(mockResponse);
    });

    it('appends query parameters correctly', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({}),
      });

      await apiClient.get('/api/test', { params: { status: 'pending', page: 1 } });

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/test?status=pending&page=1',
        expect.any(Object)
      );
    });

    it('filters out undefined params', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({}),
      });

      await apiClient.get('/api/test', { params: { status: 'pending', page: undefined } });

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/test?status=pending',
        expect.any(Object)
      );
    });

    it('makes request without query string when no params provided', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({}),
      });

      await apiClient.get('/api/test');

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/test',
        expect.any(Object)
      );
    });
  });

  describe('post', () => {
    it('makes a POST request with body', async () => {
      const requestBody = { name: 'Test Event' };
      const mockResponse = { id: 1, name: 'Test Event' };
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(mockResponse),
      });

      const result = await apiClient.post('/api/events', requestBody);

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/events',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(requestBody),
        })
      );
      expect(result).toEqual(mockResponse);
    });

    it('makes a POST request without body', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ success: true }),
      });

      await apiClient.post('/api/events/1/confirm');

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/events/1/confirm',
        expect.objectContaining({
          method: 'POST',
          body: undefined,
        })
      );
    });
  });

  describe('put', () => {
    it('makes a PUT request with body', async () => {
      const requestBody = { title: 'Updated Title' };
      const mockResponse = { id: 1, title: 'Updated Title' };
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(mockResponse),
      });

      const result = await apiClient.put('/api/events/1', requestBody);

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/events/1',
        expect.objectContaining({
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(requestBody),
        })
      );
      expect(result).toEqual(mockResponse);
    });
  });

  describe('delete', () => {
    it('makes a DELETE request', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ success: true }),
      });

      const result = await apiClient.delete('/api/channels/1');

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/channels/1',
        expect.objectContaining({
          method: 'DELETE',
        })
      );
      expect(result).toEqual({ success: true });
    });
  });

  describe('error handling', () => {
    it('throws error with message from response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        json: () => Promise.resolve({ error: 'Invalid request' }),
      });

      await expect(apiClient.get('/api/test')).rejects.toThrow('Invalid request');
    });

    it('throws error with HTTP status when no error message in response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: () => Promise.resolve({}),
      });

      await expect(apiClient.get('/api/test')).rejects.toThrow('HTTP 500');
    });

    it('throws error with HTTP status when response is not JSON', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 502,
        json: () => Promise.reject(new Error('Not JSON')),
      });

      await expect(apiClient.get('/api/test')).rejects.toThrow('HTTP 502');
    });

    it('handles network errors', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'));

      await expect(apiClient.get('/api/test')).rejects.toThrow('Network error');
    });

    it('handles timeout (abort error)', async () => {
      const abortError = new Error('Aborted');
      abortError.name = 'AbortError';
      mockFetch.mockRejectedValueOnce(abortError);

      await expect(apiClient.get('/api/test')).rejects.toThrow('Request timeout');
    });
  });

  describe('204 No Content handling', () => {
    it('returns undefined for 204 status', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 204,
      });

      const result = await apiClient.delete('/api/channels/1');

      expect(result).toBeUndefined();
    });
  });

  describe('signal/abort controller', () => {
    it('includes abort signal in request', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({}),
      });

      await apiClient.get('/api/test');

      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          signal: expect.any(AbortSignal),
        })
      );
    });
  });
});
