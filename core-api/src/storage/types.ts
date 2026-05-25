import type { AdminRole, ClaimStatus, DomainEventType } from '../domain/models.js';

export type UserIdentity = {
  chatId: number;
  maxUserId?: number;
  maxLogin?: string;
};

export type UserRecord = {
  chat_id: number;
  max_user_id?: number;
  max_login?: string;
  consent_accepted: boolean;
  consent_version?: string;
  consent_accepted_at?: number;
  notification_push: boolean;
  created_at: string;
  updated_at: string;
};

export type ClaimRecord = {
  id: number;
  chat_id: number;
  type: 'problem' | 'proposal';
  title: string;
  description: string;
  category: string;
  latitude: number;
  longitude: number;
  address_fallback?: string;
  photos: string[];
  result_media: string[];
  status: ClaimStatus;
  assignee?: string;
  official_response?: string;
  feedback?: string;
  created_at: string;
  updated_at: string;
};

export type CreateClaimInput = {
  chatId: number;
  category: string;
  description: string;
  latitude: number;
  longitude: number;
  addressFallback?: string;
  media: string[];
};

export type UpdateClaimStatusInput = {
  claimId: number;
  status: ClaimStatus;
  officialResponse?: string;
  assignee?: string;
  resultMedia?: string[];
};

export type AnnouncementRecord = {
  id: number;
  title: string;
  description: string;
  created_at: string;
  updated_at: string;
};

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

export type Invite = {
  code: string;
  phone: string;
  role: AdminRole;
  created_by: number;
  created_at: string;
  used_by?: number;
  used_at?: string;
};

export type FeedEvent = {
  id: number;
  type: 'incoming_message' | 'claim_created' | 'claim_updated' | 'status_notification';
  text: string;
  chat_id?: number;
  claim_id?: number;
  created_at: string;
  payload?: Record<string, unknown>;
};

export type DomainEventRecord = {
  id: number;
  type: DomainEventType | string;
  aggregate_type: string;
  aggregate_id?: number;
  payload: Record<string, unknown>;
  created_at: string;
  status: 'pending' | 'delivered' | 'failed' | 'dead' | string;
  attempts: number;
  next_retry_at?: string;
  last_error?: string;
  delivered_at?: string;
};

export type BotSessionRecord = {
  chat_id: number;
  flow_json: Record<string, unknown>;
  max_user_id?: number;
  max_login?: string;
  consent_accepted?: boolean;
  consent_accepted_at?: number;
  consent_version?: string;
  updated_at: number;
};

export type IncomingUpdateResult = 'processing' | 'processed' | 'failed';

export type DashboardMetrics = {
  stats: Record<string, number>;
  category_stats: Array<{ category: string; count: number }>;
  timeline: Array<{ day: string; created: number; resolved: number }>;
  queue_size: number;
  unresolved_over_7_days: number;
  sla: {
    unresolved_over_7_days: number;
    due_today: number;
    average_resolution_hours: number | null;
    p95_resolution_hours: number | null;
  };
  delivery: {
    pending: number;
    delivered: number;
    failed: number;
    dead: number;
  };
  map_points: Array<{ id: number; lat: number; lng: number; status: ClaimStatus; category: string }>;
};

export type CoreRepository = {
  initialize(): Promise<void>;
  close(): Promise<void>;
  upsertUser(identity: UserIdentity): Promise<UserRecord | null>;
  getUserByChatId(chatId: number): Promise<UserRecord | null>;
  setUserConsent(chatId: number, accepted: boolean, version: string): Promise<void>;
  setNotificationPush(chatId: number, pushEnabled: boolean): Promise<void>;
  listAnnouncements(limit: number): Promise<AnnouncementRecord[]>;
  listClaimsByChat(chatId: number, limit?: number): Promise<ClaimRecord[]>;
  listClaims(status?: ClaimStatus): Promise<ClaimRecord[]>;
  getClaimById(claimId: number): Promise<ClaimRecord | null>;
  createClaim(input: CreateClaimInput): Promise<ClaimRecord>;
  updateClaimAssignment(claimId: number, assignee: string): Promise<ClaimRecord | null>;
  updateClaimStatus(input: UpdateClaimStatusInput): Promise<{ previous: ClaimRecord; current: ClaimRecord } | null>;
  getDashboardMetrics(): Promise<DashboardMetrics>;
  seedSuperAdmins(phones: string[]): Promise<void>;
  listAdmins(): Promise<AdminUser[]>;
  loginAdminByPhone(rawPhone: string, options?: { chatId?: number; maxLogin?: string }): Promise<{ token: string; admin: AdminUser } | null>;
  getAdminByToken(token: string): Promise<AdminUser | null>;
  createInvite(actor: AdminUser, rawPhone: string, role: AdminRole): Promise<Invite>;
  activateInvite(code: string, rawPhone: string): Promise<AdminUser>;
  listInvites(): Promise<Invite[]>;
  updateAdminRole(actor: AdminUser, adminId: number, role: AdminRole): Promise<AdminUser>;
  setAdminActive(actor: AdminUser, adminId: number, active: boolean): Promise<AdminUser>;
  bindAdminChat(rawPhone: string, chatId: number, maxLogin?: string): Promise<AdminUser | null>;
  addFeedEvent(event: Omit<FeedEvent, 'id' | 'created_at'>): Promise<FeedEvent>;
  getFeed(limit: number): Promise<FeedEvent[]>;
  createDomainEvent(type: string, aggregateType: string, aggregateId: number, payload: Record<string, unknown>): Promise<DomainEventRecord>;
  listPendingDomainEvents(limit: number): Promise<DomainEventRecord[]>;
  ackDomainEvent(eventId: number): Promise<DomainEventRecord | null>;
  failDomainEvent(eventId: number, error: string, maxAttempts: number, retryBaseMs: number, retryMaxMs: number): Promise<DomainEventRecord | null>;
  claimIncomingUpdate(updateId: string, source: string, chatId?: number, messageId?: string): Promise<boolean>;
  completeIncomingUpdate(updateId: string, result: IncomingUpdateResult, error?: string): Promise<void>;
  getBotSession(chatId: number): Promise<BotSessionRecord | null>;
  saveBotSession(session: BotSessionRecord): Promise<void>;
  clearBotSession(chatId: number, updatedAt: number): Promise<BotSessionRecord>;
};
