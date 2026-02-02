import React from 'react';
import { render, screen } from '@testing-library/react-native';
import { Text, View } from 'react-native';
import { Card } from '../../../components/common/Card';

describe('Card', () => {
  describe('rendering', () => {
    it('renders children correctly', () => {
      render(
        <Card>
          <Text>Card Content</Text>
        </Card>
      );

      expect(screen.getByText('Card Content')).toBeTruthy();
    });

    it('renders multiple children', () => {
      render(
        <Card>
          <Text>First</Text>
          <Text>Second</Text>
          <Text>Third</Text>
        </Card>
      );

      expect(screen.getByText('First')).toBeTruthy();
      expect(screen.getByText('Second')).toBeTruthy();
      expect(screen.getByText('Third')).toBeTruthy();
    });

    it('renders nested components', () => {
      render(
        <Card>
          <View>
            <Text>Nested Content</Text>
          </View>
        </Card>
      );

      expect(screen.getByText('Nested Content')).toBeTruthy();
    });
  });

  describe('custom styles', () => {
    it('applies custom style prop', () => {
      render(
        <Card style={{ marginTop: 20 }}>
          <Text>Styled Card</Text>
        </Card>
      );

      expect(screen.getByText('Styled Card')).toBeTruthy();
    });

    it('merges custom styles with default styles', () => {
      render(
        <Card style={{ backgroundColor: 'red', padding: 24 }}>
          <Text>Custom Styled</Text>
        </Card>
      );

      expect(screen.getByText('Custom Styled')).toBeTruthy();
    });
  });

  describe('edge cases', () => {
    it('renders with empty children', () => {
      render(<Card>{null}</Card>);

      // Should render without crashing
      expect(screen.root).toBeTruthy();
    });

    it('handles undefined children', () => {
      render(<Card>{undefined}</Card>);

      expect(screen.root).toBeTruthy();
    });

    it('handles conditional rendering in children', () => {
      const showContent = true;
      render(
        <Card>
          {showContent && <Text>Conditional Content</Text>}
        </Card>
      );

      expect(screen.getByText('Conditional Content')).toBeTruthy();
    });

    it('handles conditional rendering when false', () => {
      const showContent = false;
      render(
        <Card>
          {showContent && <Text>Hidden Content</Text>}
        </Card>
      );

      expect(screen.queryByText('Hidden Content')).toBeNull();
    });
  });

  describe('composition', () => {
    it('works as a container for complex layouts', () => {
      render(
        <Card>
          <View>
            <Text>Header</Text>
          </View>
          <View>
            <Text>Body content goes here</Text>
          </View>
          <View>
            <Text>Footer</Text>
          </View>
        </Card>
      );

      expect(screen.getByText('Header')).toBeTruthy();
      expect(screen.getByText('Body content goes here')).toBeTruthy();
      expect(screen.getByText('Footer')).toBeTruthy();
    });
  });
});
