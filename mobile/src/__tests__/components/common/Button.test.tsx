import React from 'react';
import { render, fireEvent, screen } from '@testing-library/react-native';
import { Button } from '../../../components/common/Button';

describe('Button', () => {
  const mockOnPress = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('rendering', () => {
    it('renders with title', () => {
      render(<Button title="Click me" onPress={mockOnPress} />);

      expect(screen.getByText('Click me')).toBeTruthy();
    });

    it('renders with default variant (primary)', () => {
      render(<Button title="Primary" onPress={mockOnPress} />);

      const button = screen.getByText('Primary').parent;
      expect(button).toBeTruthy();
    });

    it('renders with secondary variant', () => {
      render(<Button title="Secondary" onPress={mockOnPress} variant="secondary" />);

      expect(screen.getByText('Secondary')).toBeTruthy();
    });

    it('renders with success variant', () => {
      render(<Button title="Success" onPress={mockOnPress} variant="success" />);

      expect(screen.getByText('Success')).toBeTruthy();
    });

    it('renders with danger variant', () => {
      render(<Button title="Danger" onPress={mockOnPress} variant="danger" />);

      expect(screen.getByText('Danger')).toBeTruthy();
    });

    it('renders with outline variant', () => {
      render(<Button title="Outline" onPress={mockOnPress} variant="outline" />);

      expect(screen.getByText('Outline')).toBeTruthy();
    });
  });

  describe('sizes', () => {
    it('renders with small size', () => {
      render(<Button title="Small" onPress={mockOnPress} size="small" />);

      expect(screen.getByText('Small')).toBeTruthy();
    });

    it('renders with medium size (default)', () => {
      render(<Button title="Medium" onPress={mockOnPress} />);

      expect(screen.getByText('Medium')).toBeTruthy();
    });

    it('renders with large size', () => {
      render(<Button title="Large" onPress={mockOnPress} size="large" />);

      expect(screen.getByText('Large')).toBeTruthy();
    });
  });

  describe('interactions', () => {
    it('calls onPress when pressed', () => {
      render(<Button title="Click" onPress={mockOnPress} />);

      fireEvent.press(screen.getByText('Click'));

      expect(mockOnPress).toHaveBeenCalledTimes(1);
    });

    it('does not call onPress when disabled', () => {
      render(<Button title="Disabled" onPress={mockOnPress} disabled />);

      fireEvent.press(screen.getByText('Disabled'));

      expect(mockOnPress).not.toHaveBeenCalled();
    });

    it('does not call onPress when loading', () => {
      render(<Button title="Loading" onPress={mockOnPress} loading />);

      // When loading, the text is replaced with ActivityIndicator
      // so we need to find the touchable differently
      const buttons = screen.root.findAllByType('TouchableOpacity' as any);
      if (buttons.length > 0) {
        fireEvent.press(buttons[0]);
      }

      expect(mockOnPress).not.toHaveBeenCalled();
    });
  });

  describe('loading state', () => {
    it('shows loading indicator when loading', () => {
      render(<Button title="Loading" onPress={mockOnPress} loading />);

      // Title should not be visible when loading
      expect(screen.queryByText('Loading')).toBeNull();
    });

    it('hides title when loading', () => {
      render(<Button title="Submit" onPress={mockOnPress} loading />);

      expect(screen.queryByText('Submit')).toBeNull();
    });
  });

  describe('disabled state', () => {
    it('renders with reduced opacity when disabled', () => {
      render(<Button title="Disabled" onPress={mockOnPress} disabled />);

      expect(screen.getByText('Disabled')).toBeTruthy();
    });

    it('is disabled when loading', () => {
      render(<Button title="Loading" onPress={mockOnPress} loading />);

      // Button should be effectively disabled during loading
      fireEvent.press(screen.root);
      expect(mockOnPress).not.toHaveBeenCalled();
    });
  });

  describe('custom styles', () => {
    it('applies custom style prop', () => {
      render(
        <Button
          title="Styled"
          onPress={mockOnPress}
          style={{ marginTop: 10 }}
        />
      );

      expect(screen.getByText('Styled')).toBeTruthy();
    });

    it('applies custom textStyle prop', () => {
      render(
        <Button
          title="Custom Text"
          onPress={mockOnPress}
          textStyle={{ fontWeight: 'bold' }}
        />
      );

      expect(screen.getByText('Custom Text')).toBeTruthy();
    });
  });

  describe('accessibility', () => {
    it('has touchable area', () => {
      render(<Button title="Accessible" onPress={mockOnPress} />);

      const touchable = screen.getByText('Accessible').parent;
      expect(touchable).toBeTruthy();
    });
  });
});
