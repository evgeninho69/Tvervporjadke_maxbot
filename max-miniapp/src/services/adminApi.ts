function resolveApiBaseUrl(): string {
  const fromEnv = (import.meta.env.VITE_API_BASE_URL || '').trim();
  if (fromEnv) return fromEnv;
  if (typeof window !== 'undefined') return window.location.origin;
  return '';
}

const API_BASE_URL = resolveApiBaseUrl();
const TOKEN_KEY = 'tver_admin_token_v2';

function trimSlash(value: string): string {
  return value.endsWith('/') ? value.slice(0, -1) : value;
}

const BASE_URL = trimSlash(API_BASE_URL);

function readMaxIdentity(): { chat_id?: number; max_login?: string } {
  if (typeof window === 'undefined') return {};
  const params = new URLSearchParams(window.location.search);
  const chatId = Number(params.get('chat_id') || '');
  const maxLogin = (params.get('max_login') || '').trim();
  return {
    chat_id: Number.isFinite(chatId) && chatId > 0 ? chatId : undefined,
    max_login: maxLogin || undefined
  };
}

export type AdminRole = 'super_admin' | 'admin' | 'operator';
export type ClaimStatus = 'moderation' | 'in_work' | 'resolved' | 'rejected';

export type AdminUser = {
  id: number;
  phone: string;
  role: AdminRole;
  active: boolean;
  created_at: string;
  updated_at: string;
  max_chat_id?: number;
  max_login?: string;
};

export type AdminClaim = {
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
  created_at: string;
  updated_at: string;
};

export type DashboardResponse = {
  role: AdminRole;
  stats: Record<string, number>;
  category_stats: Array<{ category: string; count: number }>;
  timeline: Array<{ day: string; created: number; resolved: number }>;
  queue_size: number;
  unresolved_over_7_days: number;
  sla?: {
    unresolved_over_7_days: number;
    due_today: number;
    average_resolution_hours: number | null;
    p95_resolution_hours: number | null;
  };
  delivery?: {
    pending: number;
    delivered: number;
    failed: number;
    dead: number;
  };
  latest_feed: AdminMessage[];
};

export type AdminMessage = {
  id: number;
  type: 'incoming_message' | 'claim_created' | 'claim_updated' | 'status_notification';
  text: string;
  chat_id?: number;
  claim_id?: number;
  created_at: string;
  payload?: Record<string, unknown>;
};

export type Invite = {
  code: string;
  phone: string;
  role: AdminRole;
  created_by: number;
  created_at: string;
  used_by?: number;
  used_at?: string;
};

function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) || '';
}

export function saveToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY);
}

async function request<T>(path: string, init?: RequestInit, auth = true): Promise<T> {
  const headers: Record<string, string> = {
    ...(init?.headers as Record<string, string> | undefined)
  };

  if (auth) {
    const token = getToken();
    if (!token) {
      throw new Error('Нет токена администратора');
    }
    headers.Authorization = `Bearer ${token}`;
  }

  const response = await fetch(`${BASE_URL}${path}`, {
    ...init,
    headers
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `HTTP ${response.status}`);
  }

  return (await response.json()) as T;
}

export async function loginAdmin(phone: string): Promise<{ token: string; admin: AdminUser }> {
  const identity = readMaxIdentity();
  return request<{ token: string; admin: AdminUser }>(
    '/admin/v2/auth/login',
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phone, ...identity })
    },
    false
  );
}

export async function activateInvite(code: string, phone: string): Promise<{ admin: AdminUser }> {
  return request<{ admin: AdminUser }>(
    '/admin/v2/auth/invite-activate',
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code, phone })
    },
    false
  );
}

export async function getDashboard(): Promise<DashboardResponse> {
  return request<DashboardResponse>('/admin/v2/dashboard');
}

export async function getQueue(status?: ClaimStatus): Promise<{ claims: AdminClaim[]; available_statuses: ClaimStatus[] }> {
  const query = status ? `?status=${encodeURIComponent(status)}` : '';
  return request<{ claims: AdminClaim[]; available_statuses: ClaimStatus[] }>(`/admin/v2/queue${query}`);
}

export async function getClaim(claimId: number): Promise<{ claim: AdminClaim }> {
  return request<{ claim: AdminClaim }>(`/admin/v2/claims/${claimId}`);
}

export async function updateClaimStatus(payload: {
  claimId: number;
  status: ClaimStatus;
  officialResponse?: string;
  assignee?: string;
  resultMedia?: string[];
}): Promise<{ claim: AdminClaim }> {
  return request<{ claim: AdminClaim }>(`/admin/v2/claims/${payload.claimId}/status`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      status: payload.status,
      official_response: payload.officialResponse || '',
      assignee: payload.assignee || '',
      result_media: payload.resultMedia || []
    })
  });
}

export async function assignClaim(claimId: number, assignee: string): Promise<{ claim: AdminClaim }> {
  return request<{ claim: AdminClaim }>(`/admin/v2/claims/${claimId}/assign`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ assignee })
  });
}

export async function getMessages(limit = 200): Promise<{ messages: AdminMessage[] }> {
  return request<{ messages: AdminMessage[] }>(`/admin/v2/messages?limit=${limit}`);
}

export async function getAdmins(): Promise<{ admins: AdminUser[] }> {
  return request<{ admins: AdminUser[] }>('/admin/v2/admins');
}

export async function updateAdminRole(id: number, role: AdminRole): Promise<{ admin: AdminUser }> {
  return request<{ admin: AdminUser }>(`/admin/v2/admins/${id}/role`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ role })
  });
}

export async function setAdminActive(id: number, active: boolean): Promise<{ admin: AdminUser }> {
  return request<{ admin: AdminUser }>(`/admin/v2/admins/${id}/active`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ active })
  });
}

export async function createInvite(phone: string, role: AdminRole): Promise<{ invite: Invite }> {
  return request<{ invite: Invite }>('/admin/v2/invites', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ phone, role })
  });
}

export async function getInvites(): Promise<{ invites: Invite[] }> {
  return request<{ invites: Invite[] }>('/admin/v2/invites');
}

export async function getMe(): Promise<{ admin: AdminUser }> {
  return request<{ admin: AdminUser }>('/admin/v2/me');
}

