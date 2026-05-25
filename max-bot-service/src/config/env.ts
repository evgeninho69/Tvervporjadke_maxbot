import 'dotenv/config';
import { z } from 'zod';

const envSchema = z.object({
  PORT: z.coerce.number().default(8090),
  BUILD_ID: z.string().default(`dev-${Date.now()}`),
  MAX_MODE: z.enum(['webhook', 'polling', 'hybrid']).default('polling'),
  MAX_POLL_INTERVAL_MS: z.coerce.number().default(1500),
  CORE_API_EVENTS_POLL_MS: z.coerce.number().default(4000),
  MAX_API_BASE_URL: z.string().url().default('https://platform-api.max.ru'),
  CORE_API_BASE_URL: z.string().url().default('http://localhost:8088'),
  MAX_BOT_TOKEN: z.string().min(1),
  MAX_WEBHOOK_SECRET: z.string().min(8),
  MAX_WEBHOOK_URL: z.string().url().optional(),
  CORE_API_INTERNAL_TOKEN: z.string().min(8).default('internal-dev-token'),
  MAX_MINIAPP_URL: z.string().url(),
  DEFAULT_CHANNEL_HANDLE: z.string().default('@id695005444203_biz'),
  DEFAULT_CHANNEL_NAME: z.string().default('Тверь 900'),
  SUPER_ADMIN_PHONES: z.string().default('+79105313393,+79108325238')
});

export const env = envSchema.parse(process.env);
