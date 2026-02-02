import React from 'react';
import { render, screen } from '@testing-library/react-native';
import { AttendeeChips } from '../../../components/events/AttendeeChips';
import type { Attendee } from '../../../types/event';

describe('AttendeeChips', () => {
  const mockAttendees: Attendee[] = [
    { id: 1, event_id: 1, name: 'John Doe', email: 'john@example.com' },
    { id: 2, event_id: 1, name: 'Jane Smith', email: 'jane@example.com' },
    { id: 3, event_id: 1, name: 'Bob Wilson' },
  ];

  describe('rendering', () => {
    it('renders label and attendees', () => {
      render(<AttendeeChips attendees={mockAttendees} />);

      expect(screen.getByText('Attendees:')).toBeTruthy();
      expect(screen.getByText('John Doe')).toBeTruthy();
      expect(screen.getByText('Jane Smith')).toBeTruthy();
      expect(screen.getByText('Bob Wilson')).toBeTruthy();
    });

    it('renders single attendee', () => {
      render(<AttendeeChips attendees={[mockAttendees[0]]} />);

      expect(screen.getByText('Attendees:')).toBeTruthy();
      expect(screen.getByText('John Doe')).toBeTruthy();
    });
  });

  describe('empty state', () => {
    it('returns null when attendees array is empty', () => {
      render(<AttendeeChips attendees={[]} />);

      expect(screen.queryByText('Attendees:')).toBeNull();
    });
  });

  describe('attendee chips', () => {
    it('displays attendee names in chips', () => {
      render(<AttendeeChips attendees={mockAttendees} />);

      mockAttendees.forEach((attendee) => {
        expect(screen.getByText(attendee.name)).toBeTruthy();
      });
    });

    it('handles attendees without email', () => {
      const attendeeWithoutEmail: Attendee = {
        id: 4,
        event_id: 1,
        name: 'No Email User',
      };
      render(<AttendeeChips attendees={[attendeeWithoutEmail]} />);

      expect(screen.getByText('No Email User')).toBeTruthy();
    });
  });

  describe('edge cases', () => {
    it('handles many attendees', () => {
      const manyAttendees: Attendee[] = Array.from({ length: 10 }, (_, i) => ({
        id: i + 1,
        event_id: 1,
        name: `Attendee ${i + 1}`,
      }));

      render(<AttendeeChips attendees={manyAttendees} />);

      expect(screen.getByText('Attendee 1')).toBeTruthy();
      expect(screen.getByText('Attendee 10')).toBeTruthy();
    });

    it('handles attendee with long name', () => {
      const longNameAttendee: Attendee = {
        id: 1,
        event_id: 1,
        name: 'This Is A Very Long Attendee Name That Might Overflow',
      };

      render(<AttendeeChips attendees={[longNameAttendee]} />);

      expect(screen.getByText('This Is A Very Long Attendee Name That Might Overflow')).toBeTruthy();
    });

    it('handles special characters in name', () => {
      const specialCharAttendee: Attendee = {
        id: 1,
        event_id: 1,
        name: "O'Brien & Co.",
      };

      render(<AttendeeChips attendees={[specialCharAttendee]} />);

      expect(screen.getByText("O'Brien & Co.")).toBeTruthy();
    });
  });
});
