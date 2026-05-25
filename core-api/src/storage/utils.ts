import { randomBytes } from 'node:crypto';
import { normalizeStatus, type ClaimStatus } from '../domain/models.js';
import type { ClaimRecord } from './types.js';

export function isoNow(): string {
  return new Date().toISOString();
}

export function randomToken(size = 24): string {
  return randomBytes(size).toString('hex');
}

export function normalizePhone(value: string): string {
  const digits = value.replace(/\D+/g, '');
  if (digits.length === 11 && digits.startsWith('8')) return `+7${digits.slice(1)}`;
  if (digits.length === 11 && digits.startsWith('7')) return `+${digits}`;
  if (digits.length === 10) return `+7${digits}`;
  if (value.startsWith('+') && digits.length >= 11) return `+${digits}`;
  return value.trim();
}

export function parseJsonArray(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value.filter((item) => typeof item === 'string').map((item) => String(item));
  }

  if (typeof value !== 'string' || value.trim().length === 0) return [];

  try {
    const parsed = JSON.parse(value) as unknown;
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((item) => typeof item === 'string').map((item) => String(item));
  } catch {
    return [];
  }
}

export function maybeIso(value: unknown): string {
  if (value instanceof Date) return value.toISOString();
  return String(value || '');
}

export function statusStats(claims: ClaimRecord[]): Record<string, number> {
  return {
    total: claims.length,
    moderation: claims.filter((item) => item.status === 'moderation').length,
    in_work: claims.filter((item) => item.status === 'in_work').length,
    resolved: claims.filter((item) => item.status === 'resolved').length,
    rejected: claims.filter((item) => item.status === 'rejected').length
  };
}

export function categoryStats(claims: ClaimRecord[]): Array<{ category: string; count: number }> {
  const bucket = new Map<string, number>();
  for (const claim of claims) {
    bucket.set(claim.category, (bucket.get(claim.category) || 0) + 1);
  }

  return [...bucket.entries()]
    .map(([category, count]) => ({ category, count }))
    .sort((a, b) => b.count - a.count)
    .slice(0, 8);
}

export function timelineStats(claims: ClaimRecord[]): Array<{ day: string; created: number; resolved: number }> {
  const map = new Map<string, { created: number; resolved: number }>();

  const ensure = (key: string): { created: number; resolved: number } => {
    const existing = map.get(key);
    if (existing) return existing;
    const next = { created: 0, resolved: 0 };
    map.set(key, next);
    return next;
  };

  for (const claim of claims) {
    const createdKey = claim.created_at.slice(0, 10);
    if (createdKey) ensure(createdKey).created += 1;

    if (claim.status === 'resolved') {
      const resolvedKey = claim.updated_at.slice(0, 10);
      if (resolvedKey) ensure(resolvedKey).resolved += 1;
    }
  }

  return [...map.entries()]
    .sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0))
    .slice(-14)
    .map(([day, values]) => ({ day, ...values }));
}

export function claimSlaMetrics(claims: ClaimRecord[]): {
  unresolved_over_7_days: number;
  due_today: number;
  average_resolution_hours: number | null;
  p95_resolution_hours: number | null;
} {
  const now = Date.now();
  const sevenDaysMs = 1000 * 60 * 60 * 24 * 7;
  const oneDayMs = 1000 * 60 * 60 * 24;
  const unresolved = claims.filter((claim) => claim.status !== 'resolved' && claim.status !== 'rejected');
  const resolvedHours = claims
    .filter((claim) => claim.status === 'resolved')
    .map((claim) => {
      const createdAt = new Date(claim.created_at).getTime();
      const updatedAt = new Date(claim.updated_at).getTime();
      if (!Number.isFinite(createdAt) || !Number.isFinite(updatedAt) || updatedAt < createdAt) return null;
      return (updatedAt - createdAt) / (1000 * 60 * 60);
    })
    .filter((value): value is number => typeof value === 'number');

  const average = resolvedHours.length
    ? resolvedHours.reduce((sum, value) => sum + value, 0) / resolvedHours.length
    : null;
  const sorted = [...resolvedHours].sort((a, b) => a - b);
  const p95Index = sorted.length ? Math.min(sorted.length - 1, Math.ceil(sorted.length * 0.95) - 1) : -1;

  return {
    unresolved_over_7_days: unresolved.filter((claim) => {
      const createdAt = new Date(claim.created_at).getTime();
      return Number.isFinite(createdAt) && now - createdAt > sevenDaysMs;
    }).length,
    due_today: unresolved.filter((claim) => {
      const createdAt = new Date(claim.created_at).getTime();
      if (!Number.isFinite(createdAt)) return false;
      const age = now - createdAt;
      return age >= sevenDaysMs - oneDayMs && age <= sevenDaysMs;
    }).length,
    average_resolution_hours: average === null ? null : Math.round(average * 10) / 10,
    p95_resolution_hours: p95Index >= 0 ? Math.round(sorted[p95Index] * 10) / 10 : null
  };
}

export function nextDomainEventRetryIso(attempts: number, retryBaseMs: number, retryMaxMs: number): string {
  const safeAttempts = Math.max(1, attempts);
  const cappedAttempts = Math.min(safeAttempts - 1, 8);
  const delayMs = Math.min(retryMaxMs, retryBaseMs * 2 ** cappedAttempts);
  return new Date(Date.now() + delayMs).toISOString();
}

export function mapLegacyStatus(raw: unknown): ClaimStatus {
  return normalizeStatus(String(raw || 'moderation'));
}
