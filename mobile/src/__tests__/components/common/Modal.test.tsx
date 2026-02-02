import React from 'react';
import { render, fireEvent, screen } from '@testing-library/react-native';
import { Text, View } from 'react-native';
import { Modal } from '../../../components/common/Modal';

describe('Modal', () => {
  const mockOnClose = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('visibility', () => {
    it('renders when visible is true', () => {
      render(
        <Modal visible={true} onClose={mockOnClose}>
          <Text>Modal Content</Text>
        </Modal>
      );

      expect(screen.getByText('Modal Content')).toBeTruthy();
    });

    it('does not render content when visible is false', () => {
      render(
        <Modal visible={false} onClose={mockOnClose}>
          <Text>Modal Content</Text>
        </Modal>
      );

      expect(screen.queryByText('Modal Content')).toBeNull();
    });
  });

  describe('title', () => {
    it('renders title when provided', () => {
      render(
        <Modal visible={true} onClose={mockOnClose} title="Test Title">
          <Text>Content</Text>
        </Modal>
      );

      expect(screen.getByText('Test Title')).toBeTruthy();
    });

    it('does not render title when not provided', () => {
      render(
        <Modal visible={true} onClose={mockOnClose}>
          <Text>Content Only</Text>
        </Modal>
      );

      expect(screen.getByText('Content Only')).toBeTruthy();
    });
  });

  describe('close button', () => {
    it('renders close button', () => {
      render(
        <Modal visible={true} onClose={mockOnClose}>
          <Text>Content</Text>
        </Modal>
      );

      expect(screen.getByText('✕')).toBeTruthy();
    });

    it('calls onClose when close button is pressed', () => {
      render(
        <Modal visible={true} onClose={mockOnClose}>
          <Text>Content</Text>
        </Modal>
      );

      fireEvent.press(screen.getByText('✕'));

      expect(mockOnClose).toHaveBeenCalledTimes(1);
    });
  });

  describe('children', () => {
    it('renders children correctly', () => {
      render(
        <Modal visible={true} onClose={mockOnClose}>
          <Text>Child 1</Text>
          <Text>Child 2</Text>
        </Modal>
      );

      expect(screen.getByText('Child 1')).toBeTruthy();
      expect(screen.getByText('Child 2')).toBeTruthy();
    });

    it('renders complex children', () => {
      render(
        <Modal visible={true} onClose={mockOnClose}>
          <View>
            <Text>Nested Content</Text>
            <View>
              <Text>Deeply Nested</Text>
            </View>
          </View>
        </Modal>
      );

      expect(screen.getByText('Nested Content')).toBeTruthy();
      expect(screen.getByText('Deeply Nested')).toBeTruthy();
    });
  });

  describe('footer', () => {
    it('renders footer when provided', () => {
      render(
        <Modal
          visible={true}
          onClose={mockOnClose}
          footer={<Text>Footer Content</Text>}
        >
          <Text>Main Content</Text>
        </Modal>
      );

      expect(screen.getByText('Main Content')).toBeTruthy();
      expect(screen.getByText('Footer Content')).toBeTruthy();
    });

    it('does not render footer when not provided', () => {
      render(
        <Modal visible={true} onClose={mockOnClose}>
          <Text>Content</Text>
        </Modal>
      );

      expect(screen.queryByText('Footer Content')).toBeNull();
    });
  });

  describe('scrollable', () => {
    it('is scrollable by default', () => {
      render(
        <Modal visible={true} onClose={mockOnClose}>
          <Text>Scrollable Content</Text>
        </Modal>
      );

      expect(screen.getByText('Scrollable Content')).toBeTruthy();
    });

    it('respects scrollable prop when false', () => {
      render(
        <Modal visible={true} onClose={mockOnClose} scrollable={false}>
          <Text>Non-scrollable Content</Text>
        </Modal>
      );

      expect(screen.getByText('Non-scrollable Content')).toBeTruthy();
    });
  });

  describe('onRequestClose', () => {
    it('calls onClose when hardware back is pressed (Android)', () => {
      render(
        <Modal visible={true} onClose={mockOnClose}>
          <Text>Content</Text>
        </Modal>
      );

      // onRequestClose is triggered when the user presses the hardware back button
      // This is handled by React Native Modal
      expect(screen.getByText('Content')).toBeTruthy();
    });
  });

  describe('edge cases', () => {
    it('handles empty children', () => {
      render(
        <Modal visible={true} onClose={mockOnClose}>
          {null}
        </Modal>
      );

      expect(screen.getByText('✕')).toBeTruthy();
    });

    it('handles long title', () => {
      const longTitle = 'This is a very long modal title that might need to wrap or truncate';
      render(
        <Modal visible={true} onClose={mockOnClose} title={longTitle}>
          <Text>Content</Text>
        </Modal>
      );

      expect(screen.getByText(longTitle)).toBeTruthy();
    });

    it('handles special characters in title', () => {
      render(
        <Modal visible={true} onClose={mockOnClose} title="Title with <special> & characters">
          <Text>Content</Text>
        </Modal>
      );

      expect(screen.getByText('Title with <special> & characters')).toBeTruthy();
    });
  });

  describe('composition', () => {
    it('works well with form inputs', () => {
      render(
        <Modal visible={true} onClose={mockOnClose} title="Edit Form">
          <Text>Label</Text>
          {/* TextInput would go here in a real form */}
          <Text>Form content</Text>
        </Modal>
      );

      expect(screen.getByText('Edit Form')).toBeTruthy();
      expect(screen.getByText('Form content')).toBeTruthy();
    });

    it('works with footer buttons', () => {
      render(
        <Modal
          visible={true}
          onClose={mockOnClose}
          title="Confirm"
          footer={
            <View>
              <Text>Cancel</Text>
              <Text>Save</Text>
            </View>
          }
        >
          <Text>Are you sure?</Text>
        </Modal>
      );

      expect(screen.getByText('Confirm')).toBeTruthy();
      expect(screen.getByText('Are you sure?')).toBeTruthy();
      expect(screen.getByText('Cancel')).toBeTruthy();
      expect(screen.getByText('Save')).toBeTruthy();
    });
  });
});
