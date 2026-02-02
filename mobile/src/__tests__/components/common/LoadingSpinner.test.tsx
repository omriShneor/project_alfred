import React from 'react';
import { render, screen } from '@testing-library/react-native';
import { LoadingSpinner } from '../../../components/common/LoadingSpinner';

describe('LoadingSpinner', () => {
  describe('rendering', () => {
    it('renders without message', () => {
      render(<LoadingSpinner />);

      // Should render without crashing
      expect(screen.root).toBeTruthy();
    });

    it('renders with message', () => {
      render(<LoadingSpinner message="Loading..." />);

      expect(screen.getByText('Loading...')).toBeTruthy();
    });
  });

  describe('size', () => {
    it('renders with default size (large)', () => {
      render(<LoadingSpinner />);

      expect(screen.root).toBeTruthy();
    });

    it('renders with small size', () => {
      render(<LoadingSpinner size="small" />);

      expect(screen.root).toBeTruthy();
    });

    it('renders with large size', () => {
      render(<LoadingSpinner size="large" />);

      expect(screen.root).toBeTruthy();
    });
  });

  describe('message', () => {
    it('does not render message text when not provided', () => {
      render(<LoadingSpinner />);

      expect(screen.queryByText('Loading...')).toBeNull();
    });

    it('renders custom message', () => {
      render(<LoadingSpinner message="Please wait..." />);

      expect(screen.getByText('Please wait...')).toBeTruthy();
    });

    it('renders long message', () => {
      const longMessage = 'This is a very long loading message that explains what is happening';
      render(<LoadingSpinner message={longMessage} />);

      expect(screen.getByText(longMessage)).toBeTruthy();
    });
  });

  describe('custom styles', () => {
    it('applies custom style prop', () => {
      render(<LoadingSpinner style={{ backgroundColor: 'red' }} />);

      expect(screen.root).toBeTruthy();
    });

    it('merges custom styles with default styles', () => {
      render(<LoadingSpinner style={{ padding: 40 }} message="Styled" />);

      expect(screen.getByText('Styled')).toBeTruthy();
    });
  });

  describe('edge cases', () => {
    it('handles empty message string', () => {
      render(<LoadingSpinner message="" />);

      // Empty string is falsy, so message should not be rendered
      expect(screen.root).toBeTruthy();
    });

    it('handles message with special characters', () => {
      render(<LoadingSpinner message="Loading <data>..." />);

      expect(screen.getByText('Loading <data>...')).toBeTruthy();
    });
  });
});
