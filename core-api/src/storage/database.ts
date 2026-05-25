import { existsSync, mkdirSync } from 'node:fs';
import path from 'node:path';
import { DatabaseSync } from 'node:sqlite';

export const DB_PATH = path.resolve(process.cwd(), 'data', 'tveractive.db');

let dbInstance: DatabaseSync | null = null;

function nowIso(): string {
  return new Date().toISOString();
}

function ensureDbDir(): void {
  const dir = path.dirname(DB_PATH);
  if (!existsSync(dir)) {
    mkdirSync(dir, { recursive: true });
  }
}

function hasColumn(db: DatabaseSync, table: string, column: string): boolean {
  const rows = db.prepare(`PRAGMA table_info(${table})`).all() as Array<{ name?: string }>;
  return rows.some((row) => String(row.name) === column);
}

function ensureColumn(db: DatabaseSync, table: string, column: string, ddl: string): void {
  if (!hasColumn(db, table, column)) {
    db.exec(ddl);
  }
}

function ensureBaseSchema(db: DatabaseSync): void {
  db.exec(`
    CREATE TABLE IF NOT EXISTS users (
      chat_id INTEGER PRIMARY KEY,
      max_user_id INTEGER,
      max_login TEXT,
      consent_accepted INTEGER NOT NULL DEFAULT 0,
      consent_version TEXT,
      consent_accepted_at INTEGER,
      notification_push INTEGER NOT NULL DEFAULT 1,
      created_at TEXT NOT NULL,
      updated_at TEXT NOT NULL
    );

    CREATE TABLE IF NOT EXISTS bot_sessions (
      chat_id INTEGER PRIMARY KEY,
      flow_json TEXT NOT NULL,
      max_user_id INTEGER,
      max_login TEXT,
      consent_accepted INTEGER,
      consent_accepted_at INTEGER,
      consent_version TEXT,
      updated_at INTEGER NOT NULL
    );

    CREATE TABLE IF NOT EXISTS incoming_updates (
      update_id TEXT PRIMARY KEY,
      source TEXT NOT NULL,
      chat_id INTEGER,
      message_id TEXT,
      received_at TEXT NOT NULL,
      processed_at TEXT,
      result TEXT NOT NULL DEFAULT 'processing',
      error TEXT
    );

    CREATE TABLE IF NOT EXISTS admins (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      phone TEXT NOT NULL UNIQUE,
      role TEXT NOT NULL,
      active INTEGER NOT NULL DEFAULT 1,
      max_chat_id INTEGER,
      max_login TEXT,
      created_at TEXT NOT NULL,
      updated_at TEXT NOT NULL
    );

    CREATE TABLE IF NOT EXISTS admin_sessions (
      token TEXT PRIMARY KEY,
      admin_id INTEGER NOT NULL,
      created_at TEXT NOT NULL,
      expires_at TEXT NOT NULL,
      FOREIGN KEY(admin_id) REFERENCES admins(id) ON DELETE CASCADE
    );

    CREATE TABLE IF NOT EXISTS admin_invites (
      code TEXT PRIMARY KEY,
      phone TEXT NOT NULL,
      role TEXT NOT NULL,
      created_by INTEGER NOT NULL,
      created_at TEXT NOT NULL,
      used_by INTEGER,
      used_at TEXT,
      FOREIGN KEY(created_by) REFERENCES admins(id) ON DELETE CASCADE,
      FOREIGN KEY(used_by) REFERENCES admins(id) ON DELETE SET NULL
    );

    CREATE TABLE IF NOT EXISTS admin_feed (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      type TEXT NOT NULL,
      text TEXT NOT NULL,
      chat_id INTEGER,
      claim_id INTEGER,
      created_at TEXT NOT NULL,
      payload_json TEXT
    );

    CREATE TABLE IF NOT EXISTS announcements (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      title TEXT NOT NULL,
      description TEXT NOT NULL,
      created_at TEXT NOT NULL,
      updated_at TEXT NOT NULL
    );

    CREATE TABLE IF NOT EXISTS claims (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      chat_id INTEGER NOT NULL,
      type TEXT NOT NULL,
      title TEXT NOT NULL,
      description TEXT NOT NULL,
      category TEXT NOT NULL,
      latitude REAL NOT NULL,
      longitude REAL NOT NULL,
      address_fallback TEXT,
      photos_json TEXT,
      result_media_json TEXT,
      status TEXT NOT NULL,
      assignee TEXT,
      official_response TEXT,
      feedback TEXT,
      created_at TEXT NOT NULL,
      updated_at TEXT NOT NULL,
      FOREIGN KEY(chat_id) REFERENCES users(chat_id) ON DELETE CASCADE
    );

    CREATE TABLE IF NOT EXISTS domain_events (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      type TEXT NOT NULL,
      aggregate_type TEXT NOT NULL,
      aggregate_id INTEGER,
      payload_json TEXT NOT NULL,
      created_at TEXT NOT NULL,
      status TEXT NOT NULL DEFAULT 'pending',
      attempts INTEGER NOT NULL DEFAULT 0,
      next_retry_at TEXT,
      last_error TEXT,
      delivered_at TEXT
    );

    CREATE INDEX IF NOT EXISTS idx_claims_chat ON claims(chat_id);
    CREATE INDEX IF NOT EXISTS idx_claims_status ON claims(status);
    CREATE INDEX IF NOT EXISTS idx_admin_feed_created ON admin_feed(created_at DESC);
    CREATE INDEX IF NOT EXISTS idx_domain_events_status ON domain_events(status, id);
    CREATE INDEX IF NOT EXISTS idx_domain_events_pending_due ON domain_events(status, next_retry_at, id);
    CREATE INDEX IF NOT EXISTS idx_incoming_updates_received ON incoming_updates(received_at DESC);
  `);
}

function applyMigrations(db: DatabaseSync): void {
  ensureColumn(db, 'users', 'notification_push', 'ALTER TABLE users ADD COLUMN notification_push INTEGER NOT NULL DEFAULT 1');

  ensureColumn(db, 'claims', 'address_fallback', 'ALTER TABLE claims ADD COLUMN address_fallback TEXT');
  ensureColumn(db, 'claims', 'result_media_json', 'ALTER TABLE claims ADD COLUMN result_media_json TEXT');
  ensureColumn(db, 'claims', 'assignee', 'ALTER TABLE claims ADD COLUMN assignee TEXT');
  ensureColumn(db, 'claims', 'official_response', 'ALTER TABLE claims ADD COLUMN official_response TEXT');
  ensureColumn(db, 'domain_events', 'next_retry_at', 'ALTER TABLE domain_events ADD COLUMN next_retry_at TEXT');
  ensureColumn(db, 'domain_events', 'last_error', 'ALTER TABLE domain_events ADD COLUMN last_error TEXT');

  const statusRows = db.prepare('SELECT id, status FROM claims').all() as Array<{ id: number; status: string }>;
  const updateStatus = db.prepare('UPDATE claims SET status = ? WHERE id = ?');

  for (const row of statusRows) {
    const raw = String(row.status || '').trim();
    const mapped = raw === 'pending'
      ? 'moderation'
      : raw === 'accepted'
        ? 'in_work'
        : raw === 'completed'
          ? 'resolved'
          : raw === 'declined'
            ? 'rejected'
            : raw;

    if (mapped !== raw) {
      updateStatus.run(mapped, row.id);
    }
  }
}

function seedAnnouncements(db: DatabaseSync): void {
  const countRow = db.prepare('SELECT COUNT(*) AS count FROM announcements').get() as { count?: number } | undefined;
  const announcementsCount = Number(countRow?.count ?? 0);
  if (announcementsCount > 0) return;

  const createdAt = nowIso();
  const insert = db.prepare('INSERT INTO announcements (title, description, created_at, updated_at) VALUES (?, ?, ?, ?)');
  insert.run('Новый контур запущен', 'Мини-апп и админка работают на API v2.', createdAt, createdAt);
  insert.run('Как отправить обращение', 'Нажмите кнопку «Сообщить о проблеме», прикрепите до 3 фото и укажите точку.', createdAt, createdAt);
}

export function getDb(): DatabaseSync {
  if (dbInstance) return dbInstance;

  ensureDbDir();
  const db = new DatabaseSync(DB_PATH);
  db.exec('PRAGMA journal_mode = WAL;');
  db.exec('PRAGMA foreign_keys = ON;');

  ensureBaseSchema(db);
  applyMigrations(db);
  seedAnnouncements(db);

  dbInstance = db;
  return db;
}

export function nowTimestamp(): number {
  return Date.now();
}

export function isoNow(): string {
  return nowIso();
}
