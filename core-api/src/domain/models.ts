export const CLAIM_STATUSES = ['moderation', 'in_work', 'resolved', 'rejected'] as const;

export type ClaimStatus = (typeof CLAIM_STATUSES)[number];

export const STATUS_LABELS_RU: Record<ClaimStatus, string> = {
  moderation: 'На модерации',
  in_work: 'В работе',
  resolved: 'Решено',
  rejected: 'Отклонено'
};

export type ClaimLocation = {
  lat: number;
  lng: number;
  address_fallback?: string;
};

export type ClaimMedia = {
  url: string;
};

export type NotificationSettings = {
  push_enabled: boolean;
};

export type DomainEventType = 'claim_status_changed';

export type AdminRole = 'super_admin' | 'admin' | 'operator';

export const ADMIN_ROLES: AdminRole[] = ['super_admin', 'admin', 'operator'];

export function normalizeStatus(raw: string): ClaimStatus {
  switch ((raw || '').trim()) {
    case 'pending':
    case 'moderation':
      return 'moderation';
    case 'accepted':
    case 'in_work':
      return 'in_work';
    case 'completed':
    case 'resolved':
      return 'resolved';
    case 'declined':
    case 'rejected':
      return 'rejected';
    default:
      return 'moderation';
  }
}

export function parseClaimStatus(input: unknown): ClaimStatus | null {
  const value = String(input || '').trim().toLowerCase();
  if (CLAIM_STATUSES.includes(value as ClaimStatus)) {
    return value as ClaimStatus;
  }
  return null;
}

export function statusLabel(status: ClaimStatus): string {
  return STATUS_LABELS_RU[status];
}
