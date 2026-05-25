import { env } from '../config/env.js';
import type { CoreRepository } from './types.js';
import { PostgresCoreRepository } from './postgresRepository.js';
import { SQLiteCoreRepository } from './sqliteRepository.js';

let repository: CoreRepository | null = null;

export function getRepository(): CoreRepository {
  if (repository) return repository;
  repository = env.CORE_API_STORAGE === 'postgres' ? new PostgresCoreRepository() : new SQLiteCoreRepository();
  return repository;
}
