function resolveApiBaseUrl(): string {
  const fromEnv = (import.meta.env.VITE_API_BASE_URL || '').trim();
  if (fromEnv) return fromEnv;
  if (typeof window !== 'undefined') return window.location.origin;
  return '';
}

const API_BASE_URL = resolveApiBaseUrl();

function trimSlash(value: string): string {
  return value.endsWith('/') ? value.slice(0, -1) : value;
}

const BASE_URL = trimSlash(API_BASE_URL);

export type ClaimStatus = 'moderation' | 'in_work' | 'resolved' | 'rejected';

export type Claim = {
  id: number;
  chat_id: number;
  category: string;
  title?: string;
  description: string;
  status: ClaimStatus;
  status_label: string;
  media: Array<{ url: string }>;
  result_media: Array<{ url: string }>;
  location: {
    lat: number;
    lng: number;
    address_fallback?: string;
  };
  official_response?: string;
  assignee?: string;
  created_at?: string;
  updated_at?: string;
};

export type Announcement = {
  id: number;
  title: string;
  description: string;
  created_at?: string;
  updated_at?: string;
};

export type Category = {
  id: string;
  label: string;
  icon: string;
};

export type Profile = {
  chat_id?: number;
  max_user_id?: number;
  max_login?: string;
  consent_accepted: boolean;
  notifications: {
    push_enabled: boolean;
  };
};

export type BootstrapResponse = {
  profile: Profile;
  categories: Category[];
  claims: Claim[];
  announcements: Announcement[];
};

export type CreateClaimPayload = {
  category: string;
  description: string;
  media: string[];
  location: {
    lat: number;
    lng: number;
    address_fallback?: string;
  };
};

const maxIdentity = readMaxIdentityFromUrl();

function withIdentity(path: string): string {
  const params = new URLSearchParams();

  if (maxIdentity.chatId) {
    params.set('chat_id', String(maxIdentity.chatId));
  }

  if (maxIdentity.maxUserId) {
    params.set('max_user_id', String(maxIdentity.maxUserId));
  }

  if (maxIdentity.maxLogin) {
    params.set('max_login', maxIdentity.maxLogin);
  }

  if ([...params.keys()].length === 0) {
    return path;
  }

  const separator = path.includes('?') ? '&' : '?';
  return `${path}${separator}${params.toString()}`;
}

async function requestJson<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${BASE_URL}${path}`, {
    ...init,
    headers: {
      ...(init?.headers || {})
    }
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `${path} failed: HTTP ${response.status}`);
  }

  return (await response.json()) as T;
}

export async function getBootstrap(): Promise<BootstrapResponse> {
  return requestJson<BootstrapResponse>(withIdentity('/v2/citizen/bootstrap'));
}

export async function getProfile(): Promise<{ profile: Profile }> {
  return requestJson<{ profile: Profile }>(withIdentity('/v2/citizen/profile'));
}

export async function setConsent(accepted: boolean): Promise<{ accepted: boolean }> {
  return requestJson<{ accepted: boolean }>(withIdentity('/v2/citizen/consent'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ accepted })
  });
}

export async function setNotifications(pushEnabled: boolean): Promise<{ push_enabled: boolean }> {
  return requestJson<{ push_enabled: boolean }>(withIdentity('/v2/citizen/notifications'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ push_enabled: pushEnabled })
  });
}

export async function getClaims(): Promise<{ claims: Claim[] }> {
  return requestJson<{ claims: Claim[] }>(withIdentity('/v2/citizen/claims'));
}

export async function getClaimById(claimId: number): Promise<{ claim: Claim }> {
  return requestJson<{ claim: Claim }>(withIdentity(`/v2/citizen/claims/${claimId}`));
}

export async function createClaim(payload: CreateClaimPayload): Promise<{ claim_id: number }> {
  return requestJson<{ claim_id: number }>(withIdentity('/v2/citizen/claims'), {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(payload)
  });
}

export async function getAnnouncements(): Promise<{ announcements: Announcement[] }> {
  return requestJson<{ announcements: Announcement[] }>('/v2/citizen/announcements');
}

export async function getCategories(): Promise<{ categories: Category[] }> {
  return requestJson<{ categories: Category[] }>('/v2/citizen/categories');
}

export function readMaxIdentityFromUrl(): { chatId?: number; maxLogin?: string; maxUserId?: number; initialView?: string } {
  if (typeof window === 'undefined') return {};
  const params = new URLSearchParams(window.location.search);

  const chatIdRaw = params.get('chat_id') || '';
  const maxUserIdRaw = params.get('max_user_id') || '';
  const maxLogin = params.get('max_login') || '';
  const initialView = params.get('view') || '';

  const chatId = Number(chatIdRaw);
  const maxUserId = Number(maxUserIdRaw);

  return {
    chatId: Number.isFinite(chatId) && chatId > 0 ? chatId : undefined,
    maxUserId: Number.isFinite(maxUserId) && maxUserId > 0 ? maxUserId : undefined,
    maxLogin: maxLogin || undefined,
    initialView: initialView || undefined
  };
}

