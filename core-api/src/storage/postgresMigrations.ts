export type PostgresMigration = {
  id: string;
  sql: string;
};

export const POSTGRES_MIGRATIONS: PostgresMigration[] = [
  {
    id: '001_initial_core_schema',
    sql: `
      CREATE TABLE IF NOT EXISTS schema_migrations (
        id TEXT PRIMARY KEY,
        applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
      );

      CREATE TABLE IF NOT EXISTS users (
        chat_id BIGINT PRIMARY KEY,
        max_user_id BIGINT,
        max_login TEXT,
        consent_accepted BOOLEAN NOT NULL DEFAULT FALSE,
        consent_version TEXT,
        consent_accepted_at BIGINT,
        notification_push BOOLEAN NOT NULL DEFAULT TRUE,
        created_at TIMESTAMPTZ NOT NULL,
        updated_at TIMESTAMPTZ NOT NULL
      );

      CREATE TABLE IF NOT EXISTS bot_sessions (
        chat_id BIGINT PRIMARY KEY,
        flow_json JSONB NOT NULL DEFAULT '{"step":"idle"}'::jsonb,
        max_user_id BIGINT,
        max_login TEXT,
        consent_accepted BOOLEAN,
        consent_accepted_at BIGINT,
        consent_version TEXT,
        updated_at BIGINT NOT NULL
      );

      CREATE TABLE IF NOT EXISTS incoming_updates (
        update_id TEXT PRIMARY KEY,
        source TEXT NOT NULL,
        chat_id BIGINT,
        message_id TEXT,
        received_at TIMESTAMPTZ NOT NULL,
        processed_at TIMESTAMPTZ,
        result TEXT NOT NULL DEFAULT 'processing',
        error TEXT
      );

      CREATE TABLE IF NOT EXISTS admins (
        id BIGSERIAL PRIMARY KEY,
        phone TEXT NOT NULL UNIQUE,
        role TEXT NOT NULL CHECK (role IN ('super_admin', 'admin', 'operator')),
        active BOOLEAN NOT NULL DEFAULT TRUE,
        max_chat_id BIGINT,
        max_login TEXT,
        created_at TIMESTAMPTZ NOT NULL,
        updated_at TIMESTAMPTZ NOT NULL
      );

      CREATE TABLE IF NOT EXISTS admin_sessions (
        token TEXT PRIMARY KEY,
        admin_id BIGINT NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
        created_at TIMESTAMPTZ NOT NULL,
        expires_at TIMESTAMPTZ NOT NULL
      );

      CREATE TABLE IF NOT EXISTS admin_invites (
        code TEXT PRIMARY KEY,
        phone TEXT NOT NULL,
        role TEXT NOT NULL CHECK (role IN ('super_admin', 'admin', 'operator')),
        created_by BIGINT NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
        created_at TIMESTAMPTZ NOT NULL,
        used_by BIGINT REFERENCES admins(id) ON DELETE SET NULL,
        used_at TIMESTAMPTZ
      );

      CREATE TABLE IF NOT EXISTS admin_feed (
        id BIGSERIAL PRIMARY KEY,
        type TEXT NOT NULL,
        text TEXT NOT NULL,
        chat_id BIGINT,
        claim_id BIGINT,
        created_at TIMESTAMPTZ NOT NULL,
        payload_json JSONB
      );

      CREATE TABLE IF NOT EXISTS announcements (
        id BIGSERIAL PRIMARY KEY,
        title TEXT NOT NULL,
        description TEXT NOT NULL,
        created_at TIMESTAMPTZ NOT NULL,
        updated_at TIMESTAMPTZ NOT NULL
      );

      CREATE TABLE IF NOT EXISTS claims (
        id BIGSERIAL PRIMARY KEY,
        chat_id BIGINT NOT NULL REFERENCES users(chat_id) ON DELETE CASCADE,
        type TEXT NOT NULL,
        title TEXT NOT NULL,
        description TEXT NOT NULL,
        category TEXT NOT NULL,
        latitude DOUBLE PRECISION NOT NULL,
        longitude DOUBLE PRECISION NOT NULL,
        address_fallback TEXT,
        photos_json JSONB NOT NULL DEFAULT '[]'::jsonb,
        result_media_json JSONB NOT NULL DEFAULT '[]'::jsonb,
        status TEXT NOT NULL CHECK (status IN ('moderation', 'in_work', 'resolved', 'rejected')),
        assignee TEXT,
        official_response TEXT,
        feedback TEXT,
        created_at TIMESTAMPTZ NOT NULL,
        updated_at TIMESTAMPTZ NOT NULL
      );

      CREATE TABLE IF NOT EXISTS domain_events (
        id BIGSERIAL PRIMARY KEY,
        type TEXT NOT NULL,
        aggregate_type TEXT NOT NULL,
        aggregate_id BIGINT,
        payload_json JSONB NOT NULL,
        created_at TIMESTAMPTZ NOT NULL,
        status TEXT NOT NULL DEFAULT 'pending',
        attempts INTEGER NOT NULL DEFAULT 0,
        next_retry_at TIMESTAMPTZ,
        last_error TEXT,
        delivered_at TIMESTAMPTZ
      );

      CREATE INDEX IF NOT EXISTS idx_claims_chat ON claims(chat_id);
      CREATE INDEX IF NOT EXISTS idx_claims_status ON claims(status);
      CREATE INDEX IF NOT EXISTS idx_claims_created ON claims(created_at DESC);
      CREATE INDEX IF NOT EXISTS idx_admin_feed_created ON admin_feed(created_at DESC);
      CREATE INDEX IF NOT EXISTS idx_domain_events_status ON domain_events(status, id);
      CREATE INDEX IF NOT EXISTS idx_domain_events_pending_due ON domain_events(status, next_retry_at, id);
      CREATE INDEX IF NOT EXISTS idx_incoming_updates_received ON incoming_updates(received_at DESC);
      CREATE INDEX IF NOT EXISTS idx_admin_sessions_expires ON admin_sessions(expires_at);
    `
  },
  {
    id: '002_seed_announcements',
    sql: `
      INSERT INTO announcements (title, description, created_at, updated_at)
      SELECT 'Новый контур запущен', 'Мини-апп и админка работают на API v2.', now(), now()
      WHERE NOT EXISTS (SELECT 1 FROM announcements);

      INSERT INTO announcements (title, description, created_at, updated_at)
      SELECT 'Как отправить обращение', 'Нажмите кнопку «Сообщить о проблеме», прикрепите до 3 фото и укажите точку.', now(), now()
      WHERE (SELECT COUNT(*) FROM announcements) < 2;
    `
  },
  {
    id: '003_domain_event_retry_dlq',
    sql: `
      ALTER TABLE domain_events ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ;
      ALTER TABLE domain_events ADD COLUMN IF NOT EXISTS last_error TEXT;
      CREATE INDEX IF NOT EXISTS idx_domain_events_pending_due ON domain_events(status, next_retry_at, id);
    `
  },
  {
    id: '004_incoming_update_idempotency',
    sql: `
      CREATE TABLE IF NOT EXISTS incoming_updates (
        update_id TEXT PRIMARY KEY,
        source TEXT NOT NULL,
        chat_id BIGINT,
        message_id TEXT,
        received_at TIMESTAMPTZ NOT NULL,
        processed_at TIMESTAMPTZ,
        result TEXT NOT NULL DEFAULT 'processing',
        error TEXT
      );
      CREATE INDEX IF NOT EXISTS idx_incoming_updates_received ON incoming_updates(received_at DESC);
    `
  }
];
