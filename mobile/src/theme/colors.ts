export const colors = {
  background: '#f5f5f5',
  card: '#ffffff',
  surface: '#ffffff',
  infoBackground: '#f7fbff',
  primary: '#3498db',
  success: '#27ae60',
  warning: '#f39c12',
  danger: '#e74c3c',
  dangerBackground: '#fdecea',
  text: '#333333',
  textSecondary: '#7f8c8d',
  textMuted: '#bdc3c7',
  border: '#e0e0e0',
};

export const statusColors: Record<string, string> = {
  pending: '#f39c12',
  confirmed: '#3498db',
  synced: '#27ae60',
  rejected: '#e74c3c',
  deleted: '#95a5a6',
};

export const badgeColors: Record<string, { bg: string; text: string }> = {
  sender: { bg: '#e8f5e9', text: '#2e7d32' },
  group: { bg: '#e3f2fd', text: '#1565c0' },
  create: { bg: '#d5f5e3', text: '#27ae60' },
  update: { bg: '#fef9e7', text: '#f39c12' },
  delete: { bg: '#fadbd8', text: '#e74c3c' },
};
