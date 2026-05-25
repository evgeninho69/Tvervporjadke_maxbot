import { Pool, type QueryResultRow } from 'pg';
import type { AdminRole, ClaimStatus } from '../domain/models.js';
import { env } from '../config/env.js';
import { POSTGRES_MIGRATIONS } from './postgresMigrations.js';
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
  maybeIso,
  nextDomainEventRetryIso,
  normalizePhone,
  parseJsonArray,
  randomToken,
  statusStats,
  timelineStats
} from './utils.js';

function optionalNumber(value: unknown): number | undefined {
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined;
}

function optionalString(value: unknown): string | undefined {
  const result = String(value || '').trim();
  return result || undefined;
}

function mapUser(row: QueryResultRow): UserRecord {
  return {
    chat_id: Number(row.chat_id),
    max_user_id: optionalNumber(row.max_user_id),
    max_login: optionalString(row.max_login),
    consent_accepted: Boolean(row.consent_accepted),
    consent_version: optionalString(row.consent_version),
    consent_accepted_at: optionalNumber(row.consent_accepted_at),
    notification_push: Boolean(row.notification_push),
    created_at: maybeIso(row.created_at),
    updated_at: maybeIso(row.updated_at)
  };
}

function mapClaim(row: QueryResultRow): ClaimRecord {
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
    created_at: maybeIso(row.created_at),
    updated_at: maybeIso(row.updated_at)
  };
}

function mapAnnouncement(row: QueryResultRow): AnnouncementRecord {
  return {
    id: Number(row.id),
    title: String(row.title),
    description: String(row.description),
    created_at: maybeIso(row.created_at),
    updated_at: maybeIso(row.updated_at)
  };
}

function mapAdmin(row: QueryResultRow): AdminUser {
  return {
    id: Number(row.id),
    phone: String(row.phone),
    role: String(row.role) as AdminRole,
    active: Boolean(row.active),
    created_at: maybeIso(row.created_at),
    updated_at: maybeIso(row.updated_at),
    max_chat_id: optionalNumber(row.max_chat_id),
    max_login: optionalString(row.max_login)
  };
}

function mapInvite(row: QueryResultRow): Invite {
  return {
    code: String(row.code),
    phone: String(row.phone),
    role: String(row.role) as AdminRole,
    created_by: Number(row.created_by),
    created_at: maybeIso(row.created_at),
    used_by: optionalNumber(row.used_by),
    used_at: row.used_at ? maybeIso(row.used_at) : undefined
  };
}

function mapFeed(row: QueryResultRow): FeedEvent {
  return {
    id: Number(row.id),
    type: String(row.type) as FeedEvent['type'],
    text: String(row.text),
    chat_id: optionalNumber(row.chat_id),
    claim_id: optionalNumber(row.claim_id),
    created_at: maybeIso(row.created_at),
    payload: row.payload_json ? (row.payload_json as Record<string, unknown>) : undefined
  };
}

function mapDomainEvent(row: QueryResultRow): DomainEventRecord {
  return {
    id: Number(row.id),
    type: String(row.type),
    aggregate_type: String(row.aggregate_type),
    aggregate_id: optionalNumber(row.aggregate_id),
    payload: (row.payload_json || {}) as Record<string, unknown>,
    created_at: maybeIso(row.created_at),
    status: String(row.status),
    attempts: Number(row.attempts || 0),
    next_retry_at: row.next_retry_at ? maybeIso(row.next_retry_at) : undefined,
    last_error: optionalString(row.last_error),
    delivered_at: row.delivered_at ? maybeIso(row.delivered_at) : undefined
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

function mapBotSession(row: QueryResultRow): BotSessionRecord {
  return {
    chat_id: Number(row.chat_id),
    flow_json: (row.flow_json || { step: 'idle' }) as Record<string, unknown>,
    max_user_id: optionalNumber(row.max_user_id),
    max_login: optionalString(row.max_login),
    consent_accepted: row.consent_accepted === null || row.consent_accepted === undefined ? undefined : Boolean(row.consent_accepted),
    consent_accepted_at: optionalNumber(row.consent_accepted_at),
    consent_version: optionalString(row.consent_version),
    updated_at: Number(row.updated_at || Date.now())
  };
}

export class PostgresCoreRepository implements CoreRepository {
  private readonly pool: Pool;

  constructor() {
    if (!env.DATABASE_URL) {
      throw new Error('DATABASE_URL is required when CORE_API_STORAGE=postgres');
    }

    this.pool = new Pool({
      connectionString: env.DATABASE_URL,
      ssl: env.POSTGRES_SSL ? { rejectUnauthorized: false } : undefined
    });
  }

  async initialize(): Promise<void> {
    for (const migration of POSTGRES_MIGRATIONS) {
      await this.pool.query('BEGIN');
      try {
        await this.pool.query(migration.sql);
        await this.pool.query('INSERT INTO schema_migrations (id) VALUES ($1) ON CONFLICT (id) DO NOTHING', [migration.id]);
        await this.pool.query('COMMIT');
      } catch (error) {
        await this.pool.query('ROLLBACK');
        throw error;
      }
    }
  }

  async close(): Promise<void> {
    await this.pool.end();
  }

  async upsertUser(identity: UserIdentity): Promise<UserRecord | null> {
    if (identity.chatId <= 0) return null;
    const result = await this.pool.query(
      `INSERT INTO users (chat_id, max_user_id, max_login, consent_accepted, notification_push, created_at, updated_at)
       VALUES ($1, $2, $3, FALSE, TRUE, now(), now())
       ON CONFLICT (chat_id) DO UPDATE SET
         max_user_id = COALESCE(EXCLUDED.max_user_id, users.max_user_id),
         max_login = COALESCE(EXCLUDED.max_login, users.max_login),
         updated_at = now()
       RETURNING *`,
      [identity.chatId, identity.maxUserId ?? null, identity.maxLogin ?? null]
    );
    return result.rows[0] ? mapUser(result.rows[0]) : null;
  }

  async getUserByChatId(chatId: number): Promise<UserRecord | null> {
    const result = await this.pool.query('SELECT * FROM users WHERE chat_id = $1 LIMIT 1', [chatId]);
    return result.rows[0] ? mapUser(result.rows[0]) : null;
  }

  async setUserConsent(chatId: number, accepted: boolean, version: string): Promise<void> {
    await this.pool.query(
      `UPDATE users
       SET consent_accepted = $1, consent_version = $2, consent_accepted_at = $3, updated_at = now()
       WHERE chat_id = $4`,
      [accepted, version, accepted ? Date.now() : null, chatId]
    );
  }

  async setNotificationPush(chatId: number, pushEnabled: boolean): Promise<void> {
    await this.pool.query('UPDATE users SET notification_push = $1, updated_at = now() WHERE chat_id = $2', [pushEnabled, chatId]);
  }

  async listAnnouncements(limit: number): Promise<AnnouncementRecord[]> {
    const result = await this.pool.query('SELECT * FROM announcements ORDER BY id DESC LIMIT $1', [limit]);
    return result.rows.map(mapAnnouncement);
  }

  async listClaimsByChat(chatId: number, limit = 100): Promise<ClaimRecord[]> {
    const result = await this.pool.query('SELECT * FROM claims WHERE chat_id = $1 ORDER BY id DESC LIMIT $2', [chatId, limit]);
    return result.rows.map(mapClaim);
  }

  async listClaims(status?: ClaimStatus): Promise<ClaimRecord[]> {
    const result = status
      ? await this.pool.query('SELECT * FROM claims WHERE status = $1 ORDER BY id DESC', [status])
      : await this.pool.query('SELECT * FROM claims ORDER BY id DESC');
    return result.rows.map(mapClaim);
  }

  async getClaimById(claimId: number): Promise<ClaimRecord | null> {
    const result = await this.pool.query('SELECT * FROM claims WHERE id = $1 LIMIT 1', [claimId]);
    return result.rows[0] ? mapClaim(result.rows[0]) : null;
  }

  async createClaim(input: CreateClaimInput): Promise<ClaimRecord> {
    const result = await this.pool.query(
      `INSERT INTO claims (
        chat_id, type, title, description, category, latitude, longitude, address_fallback,
        photos_json, result_media_json, status, created_at, updated_at
      ) VALUES ($1, 'problem', $2, $3, $4, $5, $6, $7, $8::jsonb, '[]'::jsonb, 'moderation', now(), now())
      RETURNING *`,
      [
        input.chatId,
        input.description.slice(0, 72),
        input.description,
        input.category,
        input.latitude,
        input.longitude,
        input.addressFallback ?? null,
        JSON.stringify(input.media)
      ]
    );
    return mapClaim(result.rows[0]);
  }

  async updateClaimAssignment(claimId: number, assignee: string): Promise<ClaimRecord | null> {
    const result = await this.pool.query('UPDATE claims SET assignee = $1, updated_at = now() WHERE id = $2 RETURNING *', [assignee, claimId]);
    return result.rows[0] ? mapClaim(result.rows[0]) : null;
  }

  async updateClaimStatus(input: UpdateClaimStatusInput): Promise<{ previous: ClaimRecord; current: ClaimRecord } | null> {
    const previous = await this.getClaimById(input.claimId);
    if (!previous) return null;

    const result = await this.pool.query(
      `UPDATE claims
       SET status = $1,
           official_response = COALESCE($2, official_response),
           feedback = COALESCE($2, feedback),
           assignee = CASE WHEN $3 != '' THEN $3 ELSE assignee END,
           result_media_json = CASE WHEN $4::jsonb != '[]'::jsonb THEN $4::jsonb ELSE result_media_json END,
           updated_at = now()
       WHERE id = $5
       RETURNING *`,
      [input.status, input.officialResponse || null, input.assignee || '', JSON.stringify(input.resultMedia || []), input.claimId]
    );

    return result.rows[0] ? { previous, current: mapClaim(result.rows[0]) } : null;
  }

  async getDashboardMetrics(): Promise<DashboardMetrics> {
    const claims = await this.listClaims();
    const sla = claimSlaMetrics(claims);
    const delivery = emptyDeliveryMetrics();
    const eventStats = await this.pool.query('SELECT status, COUNT(*) AS count FROM domain_events GROUP BY status');
    for (const row of eventStats.rows) {
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
    for (const phone of phones.map(normalizePhone).filter(Boolean)) {
      await this.pool.query(
        `INSERT INTO admins (phone, role, active, created_at, updated_at)
         VALUES ($1, 'super_admin', TRUE, now(), now())
         ON CONFLICT (phone) DO UPDATE SET role = 'super_admin', active = TRUE, updated_at = now()`,
        [phone]
      );
    }
  }

  async listAdmins(): Promise<AdminUser[]> {
    const result = await this.pool.query('SELECT * FROM admins ORDER BY id ASC');
    return result.rows.map(mapAdmin);
  }

  async loginAdminByPhone(rawPhone: string, options?: { chatId?: number; maxLogin?: string }): Promise<{ token: string; admin: AdminUser } | null> {
    const phone = normalizePhone(rawPhone);
    await this.pool.query('DELETE FROM admin_sessions WHERE expires_at <= now()');
    const found = await this.pool.query('SELECT * FROM admins WHERE phone = $1 AND active = TRUE LIMIT 1', [phone]);
    if (!found.rows[0]) return null;

    if (options?.chatId && options.chatId > 0) {
      await this.pool.query('UPDATE admins SET max_chat_id = $1, max_login = COALESCE($2, max_login), updated_at = now() WHERE id = $3', [
        options.chatId,
        options.maxLogin ?? null,
        found.rows[0].id
      ]);
    }

    const token = randomToken();
    await this.pool.query(
      "INSERT INTO admin_sessions (token, admin_id, created_at, expires_at) VALUES ($1, $2, now(), now() + interval '30 days')",
      [token, found.rows[0].id]
    );
    const fresh = await this.pool.query('SELECT * FROM admins WHERE id = $1', [found.rows[0].id]);
    return { token, admin: mapAdmin(fresh.rows[0]) };
  }

  async getAdminByToken(token: string): Promise<AdminUser | null> {
    const result = await this.pool.query(
      `SELECT a.*
       FROM admin_sessions s
       JOIN admins a ON a.id = s.admin_id
       WHERE s.token = $1 AND s.expires_at > now() AND a.active = TRUE
       LIMIT 1`,
      [token]
    );
    return result.rows[0] ? mapAdmin(result.rows[0]) : null;
  }

  async createInvite(actor: AdminUser, rawPhone: string, role: AdminRole): Promise<Invite> {
    if (actor.role !== 'super_admin') throw new Error('insufficient permissions');
    const code = randomToken(8);
    const result = await this.pool.query(
      'INSERT INTO admin_invites (code, phone, role, created_by, created_at) VALUES ($1, $2, $3, $4, now()) RETURNING *',
      [code, normalizePhone(rawPhone), role, actor.id]
    );
    return mapInvite(result.rows[0]);
  }

  async activateInvite(code: string, rawPhone: string): Promise<AdminUser> {
    const phone = normalizePhone(rawPhone);
    const client = await this.pool.connect();
    try {
      await client.query('BEGIN');
      const invite = await client.query('SELECT * FROM admin_invites WHERE code = $1 AND used_at IS NULL LIMIT 1 FOR UPDATE', [code]);
      if (!invite.rows[0]) throw new Error('invite not found');
      if (String(invite.rows[0].phone) !== phone) throw new Error('phone mismatch');

      const existing = await client.query('SELECT * FROM admins WHERE phone = $1 LIMIT 1', [phone]);
      let adminId: number;
      if (existing.rows[0]) {
        adminId = Number(existing.rows[0].id);
        await client.query('UPDATE admins SET role = $1, active = TRUE, updated_at = now() WHERE id = $2', [invite.rows[0].role, adminId]);
      } else {
        const inserted = await client.query('INSERT INTO admins (phone, role, active, created_at, updated_at) VALUES ($1, $2, TRUE, now(), now()) RETURNING id', [
          phone,
          invite.rows[0].role
        ]);
        adminId = Number(inserted.rows[0].id);
      }

      await client.query('UPDATE admin_invites SET used_by = $1, used_at = now() WHERE code = $2', [adminId, code]);
      const admin = await client.query('SELECT * FROM admins WHERE id = $1', [adminId]);
      await client.query('COMMIT');
      return mapAdmin(admin.rows[0]);
    } catch (error) {
      await client.query('ROLLBACK');
      throw error;
    } finally {
      client.release();
    }
  }

  async listInvites(): Promise<Invite[]> {
    const result = await this.pool.query('SELECT * FROM admin_invites ORDER BY created_at DESC');
    return result.rows.map(mapInvite);
  }

  async updateAdminRole(actor: AdminUser, adminId: number, role: AdminRole): Promise<AdminUser> {
    if (actor.role !== 'super_admin') throw new Error('insufficient permissions');
    const result = await this.pool.query('UPDATE admins SET role = $1, updated_at = now() WHERE id = $2 RETURNING *', [role, adminId]);
    if (!result.rows[0]) throw new Error('admin not found');
    return mapAdmin(result.rows[0]);
  }

  async setAdminActive(actor: AdminUser, adminId: number, active: boolean): Promise<AdminUser> {
    if (actor.role !== 'super_admin') throw new Error('insufficient permissions');
    const result = await this.pool.query('UPDATE admins SET active = $1, updated_at = now() WHERE id = $2 RETURNING *', [active, adminId]);
    if (!result.rows[0]) throw new Error('admin not found');
    if (!active) await this.pool.query('DELETE FROM admin_sessions WHERE admin_id = $1', [adminId]);
    return mapAdmin(result.rows[0]);
  }

  async bindAdminChat(rawPhone: string, chatId: number, maxLogin?: string): Promise<AdminUser | null> {
    const result = await this.pool.query(
      `UPDATE admins
       SET max_chat_id = $1, max_login = $2, updated_at = now()
       WHERE phone = $3 AND active = TRUE
       RETURNING *`,
      [chatId, maxLogin ?? null, normalizePhone(rawPhone)]
    );
    return result.rows[0] ? mapAdmin(result.rows[0]) : null;
  }

  async addFeedEvent(event: Omit<FeedEvent, 'id' | 'created_at'>): Promise<FeedEvent> {
    const result = await this.pool.query(
      `INSERT INTO admin_feed (type, text, chat_id, claim_id, created_at, payload_json)
       VALUES ($1, $2, $3, $4, now(), $5::jsonb)
       RETURNING *`,
      [event.type, event.text, event.chat_id ?? null, event.claim_id ?? null, event.payload ? JSON.stringify(event.payload) : null]
    );
    await this.pool.query('DELETE FROM admin_feed WHERE id NOT IN (SELECT id FROM admin_feed ORDER BY id DESC LIMIT 2000)');
    return mapFeed(result.rows[0]);
  }

  async getFeed(limit: number): Promise<FeedEvent[]> {
    const result = await this.pool.query('SELECT * FROM admin_feed ORDER BY id DESC LIMIT $1', [Math.max(1, Math.min(limit, 2000))]);
    return result.rows.map(mapFeed);
  }

  async createDomainEvent(type: string, aggregateType: string, aggregateId: number, payload: Record<string, unknown>): Promise<DomainEventRecord> {
    const result = await this.pool.query(
      `INSERT INTO domain_events (type, aggregate_type, aggregate_id, payload_json, created_at, status, attempts)
       VALUES ($1, $2, $3, $4::jsonb, now(), 'pending', 0)
       RETURNING *`,
      [type, aggregateType, aggregateId, JSON.stringify(payload)]
    );
    return mapDomainEvent(result.rows[0]);
  }

  async listPendingDomainEvents(limit: number): Promise<DomainEventRecord[]> {
    const client = await this.pool.connect();
    try {
      await client.query('BEGIN');
      const selected = await client.query(
        `SELECT * FROM domain_events
         WHERE status = 'pending'
           AND (next_retry_at IS NULL OR next_retry_at <= now())
         ORDER BY id ASC
         LIMIT $1
         FOR UPDATE SKIP LOCKED`,
        [Math.max(1, Math.min(limit, 100))]
      );
      const ids = selected.rows.map((row) => Number(row.id));
      if (ids.length > 0) {
        await client.query('UPDATE domain_events SET attempts = attempts + 1, last_error = NULL WHERE id = ANY($1::bigint[])', [ids]);
      }
      await client.query('COMMIT');
      return selected.rows.map((row) => mapDomainEvent({ ...row, attempts: Number(row.attempts || 0) + 1 }));
    } catch (error) {
      await client.query('ROLLBACK');
      throw error;
    } finally {
      client.release();
    }
  }

  async ackDomainEvent(eventId: number): Promise<DomainEventRecord | null> {
    const result = await this.pool.query(
      "UPDATE domain_events SET status = 'delivered', next_retry_at = NULL, last_error = NULL, delivered_at = now() WHERE id = $1 RETURNING *",
      [eventId]
    );
    return result.rows[0] ? mapDomainEvent(result.rows[0]) : null;
  }

  async failDomainEvent(eventId: number, error: string, maxAttempts: number, retryBaseMs: number, retryMaxMs: number): Promise<DomainEventRecord | null> {
    const found = await this.pool.query('SELECT attempts FROM domain_events WHERE id = $1 LIMIT 1', [eventId]);
    if (!found.rows[0]) return null;

    const attempts = Number(found.rows[0].attempts || 0);
    const dead = attempts >= maxAttempts;
    const nextRetryAt = dead ? null : nextDomainEventRetryIso(attempts, retryBaseMs, retryMaxMs);
    const result = await this.pool.query(
      `UPDATE domain_events
       SET status = $1,
           next_retry_at = $2,
           last_error = $3
       WHERE id = $4
       RETURNING *`,
      [dead ? 'dead' : 'pending', nextRetryAt, error.slice(0, 1000), eventId]
    );

    return result.rows[0] ? mapDomainEvent(result.rows[0]) : null;
  }

  async claimIncomingUpdate(updateId: string, source: string, chatId?: number, messageId?: string): Promise<boolean> {
    const cleanId = updateId.trim();
    if (!cleanId) return true;

    const result = await this.pool.query(
      `INSERT INTO incoming_updates (update_id, source, chat_id, message_id, received_at, result)
       VALUES ($1, $2, $3, $4, now(), 'processing')
       ON CONFLICT (update_id) DO NOTHING`,
      [cleanId, source.trim() || 'unknown', chatId ?? null, messageId ?? null]
    );

    return (result.rowCount ?? 0) > 0;
  }

  async completeIncomingUpdate(updateId: string, result: 'processing' | 'processed' | 'failed', error?: string): Promise<void> {
    const cleanId = updateId.trim();
    if (!cleanId) return;

    await this.pool.query('UPDATE incoming_updates SET result = $1, processed_at = now(), error = $2 WHERE update_id = $3', [
      result,
      error ? error.slice(0, 1000) : null,
      cleanId
    ]);
  }

  async getBotSession(chatId: number): Promise<BotSessionRecord | null> {
    const result = await this.pool.query('SELECT * FROM bot_sessions WHERE chat_id = $1 LIMIT 1', [chatId]);
    return result.rows[0] ? mapBotSession(result.rows[0]) : null;
  }

  async saveBotSession(session: BotSessionRecord): Promise<void> {
    await this.pool.query(
      `INSERT INTO bot_sessions (
        chat_id, flow_json, max_user_id, max_login, consent_accepted, consent_accepted_at, consent_version, updated_at
      ) VALUES ($1, $2::jsonb, $3, $4, $5, $6, $7, $8)
      ON CONFLICT (chat_id) DO UPDATE SET
        flow_json = EXCLUDED.flow_json,
        max_user_id = EXCLUDED.max_user_id,
        max_login = EXCLUDED.max_login,
        consent_accepted = EXCLUDED.consent_accepted,
        consent_accepted_at = EXCLUDED.consent_accepted_at,
        consent_version = EXCLUDED.consent_version,
        updated_at = EXCLUDED.updated_at`,
      [
        session.chat_id,
        JSON.stringify(session.flow_json),
        session.max_user_id ?? null,
        session.max_login ?? null,
        session.consent_accepted ?? null,
        session.consent_accepted_at ?? null,
        session.consent_version ?? null,
        session.updated_at
      ]
    );

    await this.pool.query(
      `INSERT INTO users (chat_id, max_user_id, max_login, consent_accepted, consent_version, consent_accepted_at, created_at, updated_at)
       VALUES ($1, $2, $3, COALESCE($4, FALSE), $5, $6, now(), now())
       ON CONFLICT (chat_id) DO UPDATE SET
         max_user_id = COALESCE(EXCLUDED.max_user_id, users.max_user_id),
         max_login = COALESCE(EXCLUDED.max_login, users.max_login),
         consent_accepted = COALESCE(EXCLUDED.consent_accepted, users.consent_accepted),
         consent_version = COALESCE(EXCLUDED.consent_version, users.consent_version),
         consent_accepted_at = COALESCE(EXCLUDED.consent_accepted_at, users.consent_accepted_at),
         updated_at = now()`,
      [
        session.chat_id,
        session.max_user_id ?? null,
        session.max_login ?? null,
        session.consent_accepted ?? null,
        session.consent_version ?? null,
        session.consent_accepted_at ?? null
      ]
    );
  }

  async clearBotSession(chatId: number, updatedAt: number): Promise<BotSessionRecord> {
    const session: BotSessionRecord = {
      chat_id: chatId,
      flow_json: { step: 'idle' },
      updated_at: updatedAt
    };
    await this.pool.query(
      `INSERT INTO bot_sessions (chat_id, flow_json, updated_at)
       VALUES ($1, $2::jsonb, $3)
       ON CONFLICT (chat_id) DO UPDATE SET
         flow_json = EXCLUDED.flow_json,
         max_user_id = NULL,
         max_login = NULL,
         consent_accepted = NULL,
         consent_accepted_at = NULL,
         consent_version = NULL,
         updated_at = EXCLUDED.updated_at`,
      [chatId, JSON.stringify(session.flow_json), updatedAt]
    );
    return session;
  }
}
