import 'dotenv/config';
import { z } from 'zod';

const booleanFromEnv = z.preprocess((value) => {
  if (typeof value === 'boolean') {
    return value;
  }

  if (typeof value === 'number') {
    return value !== 0;
  }

  if (typeof value === 'string') {
    const normalized = value.trim().toLowerCase();
    if (['1', 'true', 'yes', 'on'].includes(normalized)) {
      return true;
    }
    if (['0', 'false', 'no', 'off', ''].includes(normalized)) {
      return false;
    }
  }

  return value;
}, z.boolean());
const envSchema = z.object({
  PORT: z.coerce.number().default(8088),
  BUILD_ID: z.string().default(`dev-${Date.now()}`),
  CORE_API_STORAGE: z.enum(['sqlite', 'postgres']).default('sqlite'),
  CORE_API_INTERNAL_TOKEN: z.string().min(8).default('internal-dev-token'),
  DATABASE_URL: z.string().url().optional(),
  POSTGRES_SSL: booleanFromEnv.default(false),
  DOMAIN_EVENT_MAX_ATTEMPTS: z.coerce.number().int().min(1).default(8),
  DOMAIN_EVENT_RETRY_BASE_MS: z.coerce.number().int().min(100).default(5000),
  DOMAIN_EVENT_RETRY_MAX_MS: z.coerce.number().int().min(1000).default(900000),
  SUPER_ADMIN_PHONES: z.string().default('+79105313393,+79108325238'),
  MAX_MINIAPP_URL: z.string().url().default('https://tver4ever.ru/max-miniapp/')
});

export const env = envSchema.parse(process.env);

