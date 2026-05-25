import type { AdminRole, ClaimStatus } from '../domain/models.js';
import { getDb, isoNow } from './database.js';
import type {
  AdminUser,
  AnnouncementRecord,
  BotSessionRecord,
  ClaimRecord,
  CoreRepository,
  CreateClaimInput,
  DashboardMetrics,
  DomainEventRecord,
  FeedEvent,
  Invite,
  UpdateClaimStatusInput,
  UserIdentity,
  UserRecord
} from './types.js';
import {
  categoryStats,
  claimSlaMetrics,
  mapLegacyStatus,
  nextDomainEventRetryIso,
  normalizePhone,
  parseJsonArray,
  randomToken,
  statusStats,
  timelineStats
} from './utils.js';

function boolFromDb(value: unknown): boolean {
  return value === true || Number(value || 0) === 1;
}

function optionalNumber(value: unknown): number | undefined {
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined;
}

function optionalString(value: unknown): string | undefined {
  const result = String(value || '').trim();
  return result || undefined;
}

function mapUser(row: Record<string, unknown>): UserRecord {
  return {
    chat_id: Number(row.chat_id),
    max_user_id: optionalNumber(row.max_user_id),
    max_login: optionalString(row.max_login),
    consent_accepted: boolFromDb(row.consent_accepted),
    consent_version: optionalString(row.consent_version),
    consent_accepted_at: optionalNumber(row.consent_accepted_at),
    notification_push: row.notification_push === undefined ? true : boolFromDb(row.notification_push),
    created_at: String(row.created_at),
    updated_at: String(row.updated_at)
  };
}

function mapClaim(row: Record<string, unknown>): ClaimRecord {
  return {
    id: Number(row.id),
    chat_id: Number(row.chat_id),
    type: String(row.type || 'problem') as ClaimRecord['type'],
    title: String(row.title || ''),
    description: String(row.description || ''),
    category: String(row.category || 'Другое'),
    latitude: Number(row.latitude || 0),
    longitude: Number(row.longitude || 0),
    address_fallback: optionalString(row.address_fallback),
    photos: parseJsonArray(row.photos_json),
    result_media: parseJsonArray(row.result_media_json),
    status: mapLegacyStatus(row.status),
    assignee: optionalString(row.assignee),
    official_response: optionalString(row.official_response),
    feedback: optionalString(row.feedback),
    created_at: String(row.created_at),
    updated_at: String(row.updated_at)
  };
}

function mapAnnouncement(row: Record<string, unknown>): AnnouncementRecord {
  return {
    id: Number(row.id),
    title: String(row.title),
    description: String(row.description),
    created_at: String(row.created_at),
    updated_at: String(row.updated_at)
  };
}

function mapAdmin(row: Record<string, unknown>): AdminUser {
  return {
    id: Number(row.id),
    phone: String(row.phone),
    role: String(row.role) as AdminRole,
    active: boolFromDb(row.active),
    created_at: String(row.created_at),
    updated_at: String(row.updated_at),
    max_chat_id: optionalNumber(row.max_chat_id),
    max_login: optionalString(row.max_login)
  };
}

function mapInvite(row: Record<string, unknown>): Invite {
  return {
    code: String(row.code),
    phone: String(row.phone),
    role: String(row.role) as AdminRole,
    created_by: Number(row.created_by),
    created_at: String(row.created_at),
    used_by: optionalNumber(row.used_by),
    used_at: optionalString(row.used_at)
  };
}

function mapFeed(row: Record<string, unknown>): FeedEvent {
  return {
    id: Number(row.id),
    type: String(row.type) as FeedEvent['type'],
    text: String(row.text),
    chat_id: optionalNumber(row.chat_id),
    claim_id: optionalNumber(row.claim_id),
    created_at: String(row.created_at),
    payload: row.payload_json ? (JSON.parse(String(row.payload_json)) as Record<string, unknown>) : undefined
  };
}

function mapDomainEvent(row: Record<string, unknown>): DomainEventRecord {
  return {
    id: Number(row.id),
    type: String(row.type),
    aggregate_type: String(row.aggregate_type),
    aggregate_id: optionalNumber(row.aggregate_id),
    payload: JSON.parse(String(row.payload_json || '{}')) as Record<string, unknown>,
    created_at: String(row.created_at),
    status: String(row.status),
    attempts: Number(row.attempts || 0),
    next_retry_at: optionalString(row.next_retry_at),
    last_error: optionalString(row.last_error),
    delivered_at: optionalString(row.delivered_at)
  };
}

function emptyDeliveryMetrics(): DashboardMetrics['delivery'] {
  return {
    pending: 0,
    delivered: 0,
    failed: 0,
    dead: 0
  };
}

function mapBotSession(row: Record<string, unknown>): BotSessionRecord {
  return {
    chat_id: Number(row.chat_id),
    flow_json: JSON.parse(String(row.flow_json || '{"step":"idle"}')) as Record<string, unknown>,
    max_user_id: optionalNumber(row.max_user_id),
    max_login: optionalString(row.max_login),
    consent_accepted: row.consent_accepted === null || row.consent_accepted === undefined ? undefined : boolFromDb(row.consent_accepted),
    consent_accepted_at: optionalNumber(row.consent_accepted_at),
    consent_version: optionalString(row.consent_version),
    updated_at: Number(row.updated_at || Date.now())
  };
}

export class SQLiteCoreRepository implements CoreRepository {
  async initialize(): Promise<void> {
    getDb();
  }

  async close(): Promise<void> {
    return Promise.resolve();
  }

  async upsertUser(identity: UserIdentity): Promise<UserRecord | null> {
    if (identity.chatId <= 0) return null;
    const db = getDb();
    const stamp = isoNow();

    db.prepare(
      `INSERT INTO users (chat_id, max_user_id, max_login, consent_accepted, notification_push, created_at, updated_at)
       VALUES (?, ?, ?, 0, 1, ?, ?)
       ON CONFLICT(chat_id) DO UPDATE SET
         max_user_id = COALESCE(excluded.max_user_id, users.max_user_id),
         max_login = COALESCE(excluded.max_login, users.max_login),
         updated_at = excluded.updated_at`
    ).run(identity.chatId, identity.maxUserId ?? null, identity.maxLogin ?? null, stamp, stamp);

    return this.getUserByChatId(identity.chatId);
  }

  async getUserByChatId(chatId: number): Promise<UserRecord | null> {
    if (chatId <= 0) return null;
    const row = getDb().prepare('SELECT * FROM users WHERE chat_id = ? LIMIT 1').get(chatId) as Record<string, unknown> | undefined;
    return row ? mapUser(row) : null;
  }

  async setUserConsent(chatId: number, accepted: boolean, version: string): Promise<void> {
    getDb()
      .prepare('UPDATE users SET consent_accepted = ?, consent_version = ?, consent_accepted_at = ?, updated_at = ? WHERE chat_id = ?')
      .run(accepted ? 1 : 0, version, accepted ? Date.now() : null, isoNow(), chatId);
  }

  async setNotificationPush(chatId: number, pushEnabled: boolean): Promise<void> {
    getDb().prepare('UPDATE users SET notification_push = ?, updated_at = ? WHERE chat_id = ?').run(pushEnabled ? 1 : 0, isoNow(), chatId);
  }

  async listAnnouncements(limit: number): Promise<AnnouncementRecord[]> {
    const rows = getDb().prepare('SELECT * FROM announcements ORDER BY id DESC LIMIT ?').all(limit) as Record<string, unknown>[];
    return rows.map(mapAnnouncement);
  }

  async listClaimsByChat(chatId: number, limit = 100): Promise<ClaimRecord[]> {
    const rows = getDb().prepare('SELECT * FROM claims WHERE chat_id = ? ORDER BY id DESC LIMIT ?').all(chatId, limit) as Record<string, unknown>[];
    return rows.map(mapClaim);
  }

  async listClaims(status?: ClaimStatus): Promise<ClaimRecord[]> {
    const rows = getDb().prepare('SELECT * FROM claims ORDER BY id DESC').all() as Record<string, unknown>[];
    const claims = rows.map(mapClaim);
    return status ? claims.filter((claim) => claim.status === status) : claims;
  }

  async getClaimById(claimId: number): Promise<ClaimRecord | null> {
    const row = getDb().prepare('SELECT * FROM claims WHERE id = ? LIMIT 1').get(claimId) as Record<string, unknown> | undefined;
    return row ? mapClaim(row) : null;
  }

  async createClaim(input: CreateClaimInput): Promise<ClaimRecord> {
    const db = getDb();
    const stamp = isoNow();
    const title = input.description.slice(0, 72);

    db.prepare(
      `INSERT INTO claims (
        chat_id, type, title, description, category, latitude, longitude, address_fallback,
        photos_json, result_media_json, status, assignee, official_response, feedback, created_at, updated_at
      ) VALUES (?, 'problem', ?, ?, ?, ?, ?, ?, ?, '[]', 'moderation', NULL, NULL, NULL, ?, ?)`
    ).run(
      input.chatId,
      title,
      input.description,
      input.category,
      input.latitude,
      input.longitude,
      input.addressFallback ?? null,
      JSON.stringify(input.media),
      stamp,
      stamp
    );

    const inserted = db.prepare('SELECT last_insert_rowid() AS id').get() as { id?: number };
    const claim = await this.getClaimById(Number(inserted.id || 0));
    if (!claim) throw new Error('claim insert failed');
    return claim;
  }

  async updateClaimAssignment(claimId: number, assignee: string): Promise<ClaimRecord | null> {
    getDb().prepare('UPDATE claims SET assignee = ?, updated_at = ? WHERE id = ?').run(assignee, isoNow(), claimId);
    return this.getClaimById(claimId);
  }

  async updateClaimStatus(input: UpdateClaimStatusInput): Promise<{ previous: ClaimRecord; current: ClaimRecord } | null> {
    const previous = await this.getClaimById(input.claimId);
    if (!previous) return null;

    const resultMedia = JSON.stringify(input.resultMedia || []);
    getDb()
      .prepare(
        `UPDATE claims
         SET status = ?,
             official_response = COALESCE(?, official_response),
             feedback = COALESCE(?, feedback),
             assignee = CASE WHEN ? != '' THEN ? ELSE assignee END,
             result_media_json = CASE WHEN ? != '[]' THEN ? ELSE result_media_json END,
             updated_at = ?
         WHERE id = ?`
      )
      .run(
        input.status,
        input.officialResponse || null,
        input.officialResponse || null,
        input.assignee || '',
        input.assignee || '',
        resultMedia,
        resultMedia,
        isoNow(),
        input.claimId
      );

    const current = await this.getClaimById(input.claimId);
    if (!current) throw new Error('claim update failed');
    return { previous, current };
  }

  async getDashboardMetrics(): Promise<DashboardMetrics> {
    const claims = await this.listClaims();
    const sla = claimSlaMetrics(claims);
    const delivery = emptyDeliveryMetrics();
    const rows = getDb().prepare('SELECT status, COUNT(*) AS count FROM domain_events GROUP BY status').all() as Record<string, unknown>[];
    for (const row of rows) {
      const status = String(row.status);
      if (status in delivery) {
        delivery[status as keyof DashboardMetrics['delivery']] = Number(row.count || 0);
      }
    }

    return {
      stats: statusStats(claims),
      category_stats: categoryStats(claims),
      timeline: timelineStats(claims),
      queue_size: claims.filter((claim) => claim.status !== 'resolved').length,
      unresolved_over_7_days: sla.unresolved_over_7_days,
      sla,
      delivery,
      map_points: claims.map((claim) => ({
        id: claim.id,
        lat: claim.latitude,
        lng: claim.longitude,
        status: claim.status,
        category: claim.category
      }))
    };
  }

  async seedSuperAdmins(phones: string[]): Promise<void> {
    const insert = getDb().prepare(
      `INSERT INTO admins (phone, role, active, created_at, updated_at)
       VALUES (?, 'super_admin', 1, ?, ?)
       ON CONFLICT(phone) DO UPDATE SET role='super_admin', active=1, updated_at=excluded.updated_at`
    );
    const stamp = isoNow();
    for (const phone of phones.map(normalizePhone).filter(Boolean)) insert.run(phone, stamp, stamp);
  }

  async listAdmins(): Promise<AdminUser[]> {
    const rows = getDb().prepare('SELECT * FROM admins ORDER BY id ASC').all() as Record<string, unknown>[];
    return rows.map(mapAdmin);
  }

  async loginAdminByPhone(rawPhone: string, options?: { chatId?: number; maxLogin?: string }): Promise<{ token: string; admin: AdminUser } | null> {
    const db = getDb();
    const phone = normalizePhone(rawPhone);
    db.prepare('DELETE FROM admin_sessions WHERE expires_at <= ?').run(isoNow());

    const row = db.prepare('SELECT * FROM admins WHERE phone = ? AND active = 1 LIMIT 1').get(phone) as Record<string, unknown> | undefined;
    if (!row) return null;

    if (options?.chatId && options.chatId > 0) {
      db.prepare('UPDATE admins SET max_chat_id = ?, max_login = COALESCE(?, max_login), updated_at = ? WHERE id = ?').run(
        options.chatId,
        options.maxLogin ?? null,
        isoNow(),
        Number(row.id)
      );
    }

    const token = randomToken();
    const createdAt = isoNow();
    const expiresAt = new Date(Date.now() + 1000 * 60 * 60 * 24 * 30).toISOString();
    db.prepare('INSERT INTO admin_sessions (token, admin_id, created_at, expires_at) VALUES (?, ?, ?, ?)').run(token, Number(row.id), createdAt, expiresAt);

    const admin = db.prepare('SELECT * FROM admins WHERE id = ?').get(Number(row.id)) as Record<string, unknown>;
    return { token, admin: mapAdmin(admin) };
  }

  async getAdminByToken(token: string): Promise<AdminUser | null> {
    const row = getDb()
      .prepare(
        `SELECT a.*
         FROM admin_sessions s
         JOIN admins a ON a.id = s.admin_id
         WHERE s.token = ? AND s.expires_at > ? AND a.active = 1
         LIMIT 1`
      )
      .get(token, isoNow()) as Record<string, unknown> | undefined;
    return row ? mapAdmin(row) : null;
  }

  async createInvite(actor: AdminUser, rawPhone: string, role: AdminRole): Promise<Invite> {
    if (actor.role !== 'super_admin') throw new Error('insufficient permissions');
    const code = randomToken(8);
    const phone = normalizePhone(rawPhone);
    const stamp = isoNow();
    getDb().prepare('INSERT INTO admin_invites (code, phone, role, created_by, created_at) VALUES (?, ?, ?, ?, ?)').run(code, phone, role, actor.id, stamp);
    return { code, phone, role, created_by: actor.id, created_at: stamp };
  }

  async activateInvite(code: string, rawPhone: string): Promise<AdminUser> {
    const db = getDb();
    const phone = normalizePhone(rawPhone);
    const invite = db.prepare('SELECT * FROM admin_invites WHERE code = ? AND used_at IS NULL LIMIT 1').get(code) as Record<string, unknown> | undefined;
    if (!invite) throw new Error('invite not found');
    if (String(invite.phone) !== phone) throw new Error('phone mismatch');

    const existing = db.prepare('SELECT * FROM admins WHERE phone = ? LIMIT 1').get(phone) as Record<string, unknown> | undefined;
    const stamp = isoNow();
    let adminId: number;

    if (existing) {
      adminId = Number(existing.id);
      db.prepare('UPDATE admins SET role = ?, active = 1, updated_at = ? WHERE id = ?').run(String(invite.role), stamp, adminId);
    } else {
      db.prepare('INSERT INTO admins (phone, role, active, created_at, updated_at) VALUES (?, ?, 1, ?, ?)').run(phone, String(invite.role), stamp, stamp);
      const row = db.prepare('SELECT id FROM admins WHERE phone = ?').get(phone) as { id?: number };
      adminId = Number(row.id || 0);
    }

    db.prepare('UPDATE admin_invites SET used_by = ?, used_at = ? WHERE code = ?').run(adminId, stamp, code);
    const admin = db.prepare('SELECT * FROM admins WHERE id = ?').get(adminId) as Record<string, unknown>;
    return mapAdmin(admin);
  }

  async listInvites(): Promise<Invite[]> {
    const rows = getDb().prepare('SELECT * FROM admin_invites ORDER BY created_at DESC').all() as Record<string, unknown>[];
    return rows.map(mapInvite);
  }

  async updateAdminRole(actor: AdminUser, adminId: number, role: AdminRole): Promise<AdminUser> {
    if (actor.role !== 'super_admin') throw new Error('insufficient permissions');
    getDb().prepare('UPDATE admins SET role = ?, updated_at = ? WHERE id = ?').run(role, isoNow(), adminId);
    const row = getDb().prepare('SELECT * FROM admins WHERE id = ?').get(adminId) as Record<string, unknown> | undefined;
    if (!row) throw new Error('admin not found');
    return mapAdmin(row);
  }

  async setAdminActive(actor: AdminUser, adminId: number, active: boolean): Promise<AdminUser> {
    if (actor.role !== 'super_admin') throw new Error('insufficient permissions');
    const db = getDb();
    db.prepare('UPDATE admins SET active = ?, updated_at = ? WHERE id = ?').run(active ? 1 : 0, isoNow(), adminId);
    if (!active) db.prepare('DELETE FROM admin_sessions WHERE admin_id = ?').run(adminId);
    const row = db.prepare('SELECT * FROM admins WHERE id = ?').get(adminId) as Record<string, unknown> | undefined;
    if (!row) throw new Error('admin not found');
    return mapAdmin(row);
  }

  async bindAdminChat(rawPhone: string, chatId: number, maxLogin?: string): Promise<AdminUser | null> {
    const db = getDb();
    const phone = normalizePhone(rawPhone);
    const row = db.prepare('SELECT * FROM admins WHERE phone = ? AND active = 1 LIMIT 1').get(phone) as Record<string, unknown> | undefined;
    if (!row) return null;
    db.prepare('UPDATE admins SET max_chat_id = ?, max_login = ?, updated_at = ? WHERE id = ?').run(chatId, maxLogin ?? null, isoNow(), Number(row.id));
    return mapAdmin(db.prepare('SELECT * FROM admins WHERE id = ?').get(Number(row.id)) as Record<string, unknown>);
  }

  async addFeedEvent(event: Omit<FeedEvent, 'id' | 'created_at'>): Promise<FeedEvent> {
    const db = getDb();
    const stamp = isoNow();
    db.prepare('INSERT INTO admin_feed (type, text, chat_id, claim_id, created_at, payload_json) VALUES (?, ?, ?, ?, ?, ?)').run(
      event.type,
      event.text,
      event.chat_id ?? null,
      event.claim_id ?? null,
      stamp,
      event.payload ? JSON.stringify(event.payload) : null
    );
    db.prepare('DELETE FROM admin_feed WHERE id NOT IN (SELECT id FROM admin_feed ORDER BY id DESC LIMIT 2000)').run();
    const inserted = db.prepare('SELECT last_insert_rowid() AS id').get() as { id?: number };
    return { id: Number(inserted.id || 0), created_at: stamp, ...event };
  }

  async getFeed(limit: number): Promise<FeedEvent[]> {
    const rows = getDb().prepare('SELECT * FROM admin_feed ORDER BY id DESC LIMIT ?').all(Math.max(1, Math.min(limit, 2000))) as Record<string, unknown>[];
    return rows.map(mapFeed);
  }

  async createDomainEvent(type: string, aggregateType: string, aggregateId: number, payload: Record<string, unknown>): Promise<DomainEventRecord> {
    const db = getDb();
    const stamp = isoNow();
    db.prepare(
      `INSERT INTO domain_events (type, aggregate_type, aggregate_id, payload_json, created_at, status, attempts)
       VALUES (?, ?, ?, ?, ?, 'pending', 0)`
    ).run(type, aggregateType, aggregateId, JSON.stringify(payload), stamp);
    const inserted = db.prepare('SELECT last_insert_rowid() AS id').get() as { id?: number };
    const row = db.prepare('SELECT * FROM domain_events WHERE id = ?').get(Number(inserted.id || 0)) as Record<string, unknown>;
    return mapDomainEvent(row);
  }

  async listPendingDomainEvents(limit: number): Promise<DomainEventRecord[]> {
    const db = getDb();
    const rows = db
      .prepare("SELECT * FROM domain_events WHERE status = 'pending' AND (next_retry_at IS NULL OR next_retry_at <= ?) ORDER BY id ASC LIMIT ?")
      .all(isoNow(), Math.max(1, Math.min(limit, 100))) as Record<string, unknown>[];
    const mark = db.prepare("UPDATE domain_events SET attempts = attempts + 1, last_error = NULL WHERE id = ?");
    for (const row of rows) mark.run(Number(row.id));
    return rows.map((row) => mapDomainEvent({ ...row, attempts: Number(row.attempts || 0) + 1 }));
  }

  async ackDomainEvent(eventId: number): Promise<DomainEventRecord | null> {
    const db = getDb();
    db.prepare("UPDATE domain_events SET status = 'delivered', next_retry_at = NULL, last_error = NULL, delivered_at = ? WHERE id = ?").run(isoNow(), eventId);
    const row = db.prepare('SELECT * FROM domain_events WHERE id = ? LIMIT 1').get(eventId) as Record<string, unknown> | undefined;
    return row ? mapDomainEvent(row) : null;
  }

  async failDomainEvent(eventId: number, error: string, maxAttempts: number, retryBaseMs: number, retryMaxMs: number): Promise<DomainEventRecord | null> {
    const db = getDb();
    const row = db.prepare('SELECT * FROM domain_events WHERE id = ? LIMIT 1').get(eventId) as Record<string, unknown> | undefined;
    if (!row) return null;

    const attempts = Number(row.attempts || 0);
    const dead = attempts >= maxAttempts;
    const nextRetryAt = dead ? null : nextDomainEventRetryIso(attempts, retryBaseMs, retryMaxMs);

    db.prepare('UPDATE domain_events SET status = ?, next_retry_at = ?, last_error = ? WHERE id = ?').run(
      dead ? 'dead' : 'pending',
      nextRetryAt,
      error.slice(0, 1000),
      eventId
    );

    const updated = db.prepare('SELECT * FROM domain_events WHERE id = ? LIMIT 1').get(eventId) as Record<string, unknown> | undefined;
    return updated ? mapDomainEvent(updated) : null;
  }

  async claimIncomingUpdate(updateId: string, source: string, chatId?: number, messageId?: string): Promise<boolean> {
    const cleanId = updateId.trim();
    if (!cleanId) return true;

    const result = getDb()
      .prepare(
        `INSERT OR IGNORE INTO incoming_updates (update_id, source, chat_id, message_id, received_at, result)
         VALUES (?, ?, ?, ?, ?, 'processing')`
      )
      .run(cleanId, source.trim() || 'unknown', chatId ?? null, messageId ?? null, isoNow());

    return Number(result.changes || 0) > 0;
  }

  async completeIncomingUpdate(updateId: string, result: 'processing' | 'processed' | 'failed', error?: string): Promise<void> {
    const cleanId = updateId.trim();
    if (!cleanId) return;

    getDb()
      .prepare('UPDATE incoming_updates SET result = ?, processed_at = ?, error = ? WHERE update_id = ?')
      .run(result, isoNow(), error ? error.slice(0, 1000) : null, cleanId);
  }

  async getBotSession(chatId: number): Promise<BotSessionRecord | null> {
    const row = getDb().prepare('SELECT * FROM bot_sessions WHERE chat_id = ? LIMIT 1').get(chatId) as Record<string, unknown> | undefined;
    return row ? mapBotSession(row) : null;
  }

  async saveBotSession(session: BotSessionRecord): Promise<void> {
    const db = getDb();
    db.prepare(
      `INSERT INTO bot_sessions (
        chat_id, flow_json, max_user_id, max_login, consent_accepted, consent_accepted_at, consent_version, updated_at
      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
      ON CONFLICT(chat_id) DO UPDATE SET
        flow_json=excluded.flow_json,
        max_user_id=excluded.max_user_id,
        max_login=excluded.max_login,
        consent_accepted=excluded.consent_accepted,
        consent_accepted_at=excluded.consent_accepted_at,
        consent_version=excluded.consent_version,
        updated_at=excluded.updated_at`
    ).run(
      session.chat_id,
      JSON.stringify(session.flow_json),
      session.max_user_id ?? null,
      session.max_login ?? null,
      session.consent_accepted === undefined ? null : session.consent_accepted ? 1 : 0,
      session.consent_accepted_at ?? null,
      session.consent_version ?? null,
      session.updated_at
    );

    db.prepare(
      `INSERT INTO users (
        chat_id, max_user_id, max_login, consent_accepted, consent_version, consent_accepted_at, created_at, updated_at
      ) VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
      ON CONFLICT(chat_id) DO UPDATE SET
        max_user_id=COALESCE(excluded.max_user_id, users.max_user_id),
        max_login=COALESCE(excluded.max_login, users.max_login),
        consent_accepted=COALESCE(excluded.consent_accepted, users.consent_accepted),
        consent_version=COALESCE(excluded.consent_version, users.consent_version),
        consent_accepted_at=COALESCE(excluded.consent_accepted_at, users.consent_accepted_at),
        updated_at=datetime('now')`
    ).run(
      session.chat_id,
      session.max_user_id ?? null,
      session.max_login ?? null,
      session.consent_accepted === undefined ? 0 : session.consent_accepted ? 1 : 0,
      session.consent_version ?? null,
      session.consent_accepted_at ?? null
    );
  }

  async clearBotSession(chatId: number, updatedAt: number): Promise<BotSessionRecord> {
    const session: BotSessionRecord = {
      chat_id: chatId,
      flow_json: { step: 'idle' },
      updated_at: updatedAt
    };
    getDb()
      .prepare(
        `INSERT INTO bot_sessions (chat_id, flow_json, updated_at)
         VALUES (?, ?, ?)
         ON CONFLICT(chat_id) DO UPDATE SET
           flow_json=excluded.flow_json,
           max_user_id=NULL,
           max_login=NULL,
           consent_accepted=NULL,
           consent_accepted_at=NULL,
           consent_version=NULL,
           updated_at=excluded.updated_at`
      )
      .run(chatId, JSON.stringify(session.flow_json), updatedAt);
    return session;
  }
}
