import 'dotenv/config';
import { existsSync } from 'node:fs';
import path from 'node:path';
import { DatabaseSync } from 'node:sqlite';
import { Pool } from 'pg';
import { POSTGRES_MIGRATIONS } from '../src/storage/postgresMigrations.js';

type Row = Record<string, unknown>;

const sqlitePath = path.resolve(process.cwd(), process.env.SQLITE_DB_PATH || 'data/tveractive.db');
const databaseUrl = process.env.DATABASE_URL;

if (!databaseUrl) {
  throw new Error('DATABASE_URL is required');
}

if (!existsSync(sqlitePath)) {
  throw new Error(`SQLite database not found: ${sqlitePath}`);
}

const sqlite = new DatabaseSync(sqlitePath, { readOnly: true });
const pool = new Pool({
  connectionString: databaseUrl,
  ssl: process.env.POSTGRES_SSL === 'true' ? { rejectUnauthorized: false } : undefined
});

function tableExists(table: string): boolean {
  const row = sqlite.prepare("SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?").get(table) as Row | undefined;
  return Boolean(row);
}

function rows(table: string): Row[] {
  if (!tableExists(table)) return [];
  return sqlite.prepare(`SELECT * FROM ${table}`).all() as Row[];
}

function jsonValue(value: unknown, fallback: unknown): string {
  if (typeof value !== 'string' || !value.trim()) return JSON.stringify(fallback);
  try {
    return JSON.stringify(JSON.parse(value));
  } catch {
    return JSON.stringify(fallback);
  }
}

function boolValue(value: unknown): boolean {
  return value === true || Number(value || 0) === 1;
}

async function runMigrations(): Promise<void> {
  for (const migration of POSTGRES_MIGRATIONS) {
    await pool.query('BEGIN');
    try {
      await pool.query(migration.sql);
      await pool.query('INSERT INTO schema_migrations (id) VALUES ($1) ON CONFLICT (id) DO NOTHING', [migration.id]);
      await pool.query('COMMIT');
    } catch (error) {
      await pool.query('ROLLBACK');
      throw error;
    }
  }
}

async function resetSequence(table: string, column = 'id'): Promise<void> {
  await pool.query(
    `SELECT setval(
      pg_get_serial_sequence($1, $2),
      COALESCE((SELECT MAX(${column}) FROM ${table}), 1),
      (SELECT COUNT(*) FROM ${table}) > 0
    )`,
    [table, column]
  );
}

async function migrateUsers(): Promise<number> {
  let count = 0;
  for (const row of rows('users')) {
    await pool.query(
      `INSERT INTO users (
        chat_id, max_user_id, max_login, consent_accepted, consent_version, consent_accepted_at,
        notification_push, created_at, updated_at
      ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8::timestamptz, $9::timestamptz)
      ON CONFLICT (chat_id) DO UPDATE SET
        max_user_id = EXCLUDED.max_user_id,
        max_login = EXCLUDED.max_login,
        consent_accepted = EXCLUDED.consent_accepted,
        consent_version = EXCLUDED.consent_version,
        consent_accepted_at = EXCLUDED.consent_accepted_at,
        notification_push = EXCLUDED.notification_push,
        updated_at = EXCLUDED.updated_at`,
      [
        row.chat_id,
        row.max_user_id ?? null,
        row.max_login ?? null,
        boolValue(row.consent_accepted),
        row.consent_version ?? null,
        row.consent_accepted_at ?? null,
        row.notification_push === undefined ? true : boolValue(row.notification_push),
        row.created_at,
        row.updated_at
      ]
    );
    count += 1;
  }
  return count;
}

async function migrateBotSessions(): Promise<number> {
  let count = 0;
  for (const row of rows('bot_sessions')) {
    await pool.query(
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
        row.chat_id,
        jsonValue(row.flow_json, { step: 'idle' }),
        row.max_user_id ?? null,
        row.max_login ?? null,
        row.consent_accepted === null || row.consent_accepted === undefined ? null : boolValue(row.consent_accepted),
        row.consent_accepted_at ?? null,
        row.consent_version ?? null,
        row.updated_at
      ]
    );
    count += 1;
  }
  return count;
}

async function migrateIncomingUpdates(): Promise<number> {
  let count = 0;
  for (const row of rows('incoming_updates')) {
    await pool.query(
      `INSERT INTO incoming_updates (update_id, source, chat_id, message_id, received_at, processed_at, result, error)
       VALUES ($1, $2, $3, $4, $5::timestamptz, $6::timestamptz, $7, $8)
       ON CONFLICT (update_id) DO UPDATE SET
         source = EXCLUDED.source,
         chat_id = EXCLUDED.chat_id,
         message_id = EXCLUDED.message_id,
         processed_at = EXCLUDED.processed_at,
         result = EXCLUDED.result,
         error = EXCLUDED.error`,
      [
        row.update_id,
        row.source ?? 'unknown',
        row.chat_id ?? null,
        row.message_id ?? null,
        row.received_at,
        row.processed_at ?? null,
        row.result ?? 'processed',
        row.error ?? null
      ]
    );
    count += 1;
  }
  return count;
}

async function migrateAdmins(): Promise<number> {
  let count = 0;
  for (const row of rows('admins')) {
    await pool.query(
      `INSERT INTO admins (id, phone, role, active, max_chat_id, max_login, created_at, updated_at)
       VALUES ($1, $2, $3, $4, $5, $6, $7::timestamptz, $8::timestamptz)
       ON CONFLICT (phone) DO UPDATE SET
         role = EXCLUDED.role,
         active = EXCLUDED.active,
         max_chat_id = EXCLUDED.max_chat_id,
         max_login = EXCLUDED.max_login,
         updated_at = EXCLUDED.updated_at`,
      [
        row.id,
        row.phone,
        row.role,
        boolValue(row.active),
        row.max_chat_id ?? null,
        row.max_login ?? null,
        row.created_at,
        row.updated_at
      ]
    );
    count += 1;
  }
  if (count > 0) await resetSequence('admins');
  return count;
}

async function migrateAdminSessions(): Promise<number> {
  let count = 0;
  for (const row of rows('admin_sessions')) {
    await pool.query(
      `INSERT INTO admin_sessions (token, admin_id, created_at, expires_at)
       VALUES ($1, $2, $3::timestamptz, $4::timestamptz)
       ON CONFLICT (token) DO UPDATE SET
         admin_id = EXCLUDED.admin_id,
         created_at = EXCLUDED.created_at,
         expires_at = EXCLUDED.expires_at`,
      [row.token, row.admin_id, row.created_at, row.expires_at]
    );
    count += 1;
  }
  return count;
}

async function migrateInvites(): Promise<number> {
  let count = 0;
  for (const row of rows('admin_invites')) {
    await pool.query(
      `INSERT INTO admin_invites (code, phone, role, created_by, created_at, used_by, used_at)
       VALUES ($1, $2, $3, $4, $5::timestamptz, $6, $7::timestamptz)
       ON CONFLICT (code) DO UPDATE SET
         phone = EXCLUDED.phone,
         role = EXCLUDED.role,
         used_by = EXCLUDED.used_by,
         used_at = EXCLUDED.used_at`,
      [row.code, row.phone, row.role, row.created_by, row.created_at, row.used_by ?? null, row.used_at ?? null]
    );
    count += 1;
  }
  return count;
}

async function migrateFeed(): Promise<number> {
  let count = 0;
  for (const row of rows('admin_feed')) {
    await pool.query(
      `INSERT INTO admin_feed (id, type, text, chat_id, claim_id, created_at, payload_json)
       VALUES ($1, $2, $3, $4, $5, $6::timestamptz, $7::jsonb)
       ON CONFLICT (id) DO UPDATE SET
         type = EXCLUDED.type,
         text = EXCLUDED.text,
         chat_id = EXCLUDED.chat_id,
         claim_id = EXCLUDED.claim_id,
         payload_json = EXCLUDED.payload_json`,
      [row.id, row.type, row.text, row.chat_id ?? null, row.claim_id ?? null, row.created_at, jsonValue(row.payload_json, null)]
    );
    count += 1;
  }
  if (count > 0) await resetSequence('admin_feed');
  return count;
}

async function migrateAnnouncements(): Promise<number> {
  let count = 0;
  for (const row of rows('announcements')) {
    await pool.query(
      `INSERT INTO announcements (id, title, description, created_at, updated_at)
       VALUES ($1, $2, $3, $4::timestamptz, $5::timestamptz)
       ON CONFLICT (id) DO UPDATE SET
         title = EXCLUDED.title,
         description = EXCLUDED.description,
         updated_at = EXCLUDED.updated_at`,
      [row.id, row.title, row.description, row.created_at, row.updated_at]
    );
    count += 1;
  }
  if (count > 0) await resetSequence('announcements');
  return count;
}

async function migrateClaims(): Promise<number> {
  let count = 0;
  for (const row of rows('claims')) {
    await pool.query(
      `INSERT INTO claims (
        id, chat_id, type, title, description, category, latitude, longitude, address_fallback,
        photos_json, result_media_json, status, assignee, official_response, feedback, created_at, updated_at
      ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11::jsonb, $12, $13, $14, $15, $16::timestamptz, $17::timestamptz)
      ON CONFLICT (id) DO UPDATE SET
        chat_id = EXCLUDED.chat_id,
        type = EXCLUDED.type,
        title = EXCLUDED.title,
        description = EXCLUDED.description,
        category = EXCLUDED.category,
        latitude = EXCLUDED.latitude,
        longitude = EXCLUDED.longitude,
        address_fallback = EXCLUDED.address_fallback,
        photos_json = EXCLUDED.photos_json,
        result_media_json = EXCLUDED.result_media_json,
        status = EXCLUDED.status,
        assignee = EXCLUDED.assignee,
        official_response = EXCLUDED.official_response,
        feedback = EXCLUDED.feedback,
        updated_at = EXCLUDED.updated_at`,
      [
        row.id,
        row.chat_id,
        row.type ?? 'problem',
        row.title ?? '',
        row.description ?? '',
        row.category ?? 'other',
        row.latitude ?? 0,
        row.longitude ?? 0,
        row.address_fallback ?? null,
        jsonValue(row.photos_json, []),
        jsonValue(row.result_media_json, []),
        row.status ?? 'moderation',
        row.assignee ?? null,
        row.official_response ?? null,
        row.feedback ?? null,
        row.created_at,
        row.updated_at
      ]
    );
    count += 1;
  }
  if (count > 0) await resetSequence('claims');
  return count;
}

async function migrateDomainEvents(): Promise<number> {
  let count = 0;
  for (const row of rows('domain_events')) {
    await pool.query(
      `INSERT INTO domain_events (
        id, type, aggregate_type, aggregate_id, payload_json, created_at, status, attempts,
        next_retry_at, last_error, delivered_at
      ) VALUES ($1, $2, $3, $4, $5::jsonb, $6::timestamptz, $7, $8, $9::timestamptz, $10, $11::timestamptz)
      ON CONFLICT (id) DO UPDATE SET
        type = EXCLUDED.type,
        aggregate_type = EXCLUDED.aggregate_type,
        aggregate_id = EXCLUDED.aggregate_id,
        payload_json = EXCLUDED.payload_json,
        status = EXCLUDED.status,
        attempts = EXCLUDED.attempts,
        next_retry_at = EXCLUDED.next_retry_at,
        last_error = EXCLUDED.last_error,
        delivered_at = EXCLUDED.delivered_at`,
      [
        row.id,
        row.type,
        row.aggregate_type,
        row.aggregate_id ?? null,
        jsonValue(row.payload_json, {}),
        row.created_at,
        row.status ?? 'pending',
        row.attempts ?? 0,
        row.next_retry_at ?? null,
        row.last_error ?? null,
        row.delivered_at ?? null
      ]
    );
    count += 1;
  }
  if (count > 0) await resetSequence('domain_events');
  return count;
}

async function main(): Promise<void> {
  console.log(JSON.stringify({ level: 'info', message: 'migration started', sqlitePath }));
  await runMigrations();

  const summary = {
    users: await migrateUsers(),
    bot_sessions: await migrateBotSessions(),
    incoming_updates: await migrateIncomingUpdates(),
    admins: await migrateAdmins(),
    admin_sessions: await migrateAdminSessions(),
    admin_invites: await migrateInvites(),
    admin_feed: await migrateFeed(),
    announcements: await migrateAnnouncements(),
    claims: await migrateClaims(),
    domain_events: await migrateDomainEvents()
  };

  console.log(JSON.stringify({ level: 'info', message: 'migration completed', summary }));
}

main()
  .catch((error) => {
    console.error(JSON.stringify({ level: 'error', message: error instanceof Error ? error.message : 'unknown_error' }));
    process.exitCode = 1;
  })
  .finally(async () => {
    sqlite.close();
    await pool.end();
  });
