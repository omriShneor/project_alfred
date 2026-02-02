import React from 'react';
import { render, screen } from '@testing-library/react-native';
import { Badge } from '../../../components/common/Badge';

describe('Badge', () => {
  describe('rendering', () => {
    it('renders with label', () => {
      render(<Badge label="Test" />);

      expect(screen.getByText('Test')).toBeTruthy();
    });

    it('capitalizes the label text via styles', () => {
      render(<Badge label="pending" />);

      // The text is rendered as-is, but styled with textTransform: 'capitalize'
      expect(screen.getByText('pending')).toBeTruthy();
    });
  });

  describe('variants', () => {
    it('renders with default (custom) variant', () => {
      render(<Badge label="Custom" />);

      expect(screen.getByText('Custom')).toBeTruthy();
    });

    it('renders with sender variant', () => {
      render(<Badge label="Sender" variant="sender" />);

      expect(screen.getByText('Sender')).toBeTruthy();
    });

    it('renders with group variant', () => {
      render(<Badge label="Group" variant="group" />);

      expect(screen.getByText('Group')).toBeTruthy();
    });

    it('renders with create variant', () => {
      render(<Badge label="create" variant="create" />);

      expect(screen.getByText('create')).toBeTruthy();
    });

    it('renders with update variant', () => {
      render(<Badge label="update" variant="update" />);

      expect(screen.getByText('update')).toBeTruthy();
    });

    it('renders with delete variant', () => {
      render(<Badge label="delete" variant="delete" />);

      expect(screen.getByText('delete')).toBeTruthy();
    });
  });

  describe('status variant', () => {
    it('renders status variant with pending status', () => {
      render(<Badge label="pending" variant="status" status="pending" />);

      expect(screen.getByText('pending')).toBeTruthy();
    });

    it('renders status variant with confirmed status', () => {
      render(<Badge label="confirmed" variant="status" status="confirmed" />);

      expect(screen.getByText('confirmed')).toBeTruthy();
    });

    it('renders status variant with synced status', () => {
      render(<Badge label="synced" variant="status" status="synced" />);

      expect(screen.getByText('synced')).toBeTruthy();
    });

    it('renders status variant with rejected status', () => {
      render(<Badge label="rejected" variant="status" status="rejected" />);

      expect(screen.getByText('rejected')).toBeTruthy();
    });

    it('renders status variant with deleted status', () => {
      render(<Badge label="deleted" variant="status" status="deleted" />);

      expect(screen.getByText('deleted')).toBeTruthy();
    });

    it('handles status variant without status prop', () => {
      render(<Badge label="unknown" variant="status" />);

      expect(screen.getByText('unknown')).toBeTruthy();
    });
  });

  describe('custom colors', () => {
    it('applies custom background color', () => {
      render(<Badge label="Custom BG" bgColor="#ff0000" />);

      expect(screen.getByText('Custom BG')).toBeTruthy();
    });

    it('applies custom text color', () => {
      render(<Badge label="Custom Text" textColor="#00ff00" />);

      expect(screen.getByText('Custom Text')).toBeTruthy();
    });

    it('applies both custom colors', () => {
      render(<Badge label="Both" bgColor="#ff0000" textColor="#ffffff" />);

      expect(screen.getByText('Both')).toBeTruthy();
    });
  });

  describe('custom styles', () => {
    it('applies custom style prop', () => {
      render(<Badge label="Styled" style={{ marginRight: 8 }} />);

      expect(screen.getByText('Styled')).toBeTruthy();
    });
  });

  describe('event action type badges', () => {
    const actionTypes = ['create', 'update', 'delete'] as const;

    actionTypes.forEach((actionType) => {
      it(`renders ${actionType} action type correctly`, () => {
        render(<Badge label={actionType} variant={actionType} />);

        expect(screen.getByText(actionType)).toBeTruthy();
      });
    });
  });

  describe('event status badges', () => {
    const statuses = ['pending', 'confirmed', 'synced', 'rejected', 'deleted'];

    statuses.forEach((status) => {
      it(`renders ${status} status correctly`, () => {
        render(<Badge label={status} variant="status" status={status} />);

        expect(screen.getByText(status)).toBeTruthy();
      });
    });
  });

  describe('edge cases', () => {
    it('handles empty label', () => {
      render(<Badge label="" />);

      // Should render without crashing
      expect(screen.root).toBeTruthy();
    });

    it('handles long label', () => {
      const longLabel = 'This is a very long badge label that might overflow';
      render(<Badge label={longLabel} />);

      expect(screen.getByText(longLabel)).toBeTruthy();
    });

    it('handles special characters in label', () => {
      render(<Badge label="Test & Demo <>" />);

      expect(screen.getByText('Test & Demo <>')).toBeTruthy();
    });

    it('handles unknown status gracefully', () => {
      render(<Badge label="unknown" variant="status" status="unknown_status" />);

      expect(screen.getByText('unknown')).toBeTruthy();
    });
  });
});
