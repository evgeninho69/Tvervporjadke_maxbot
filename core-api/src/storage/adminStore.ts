import { env } from '../config/env.js';
import type { AdminRole } from '../domain/models.js';
import type { AdminUser, FeedEvent, Invite } from './types.js';
import { getRepository } from './repository.js';
import { normalizePhone } from './utils.js';

export class AdminStore {
  private seeded = false;

  private superAdminPhones(): string[] {
    return env.SUPER_ADMIN_PHONES.split(',').map(normalizePhone).filter(Boolean);
  }

  private async seedIfNeeded(): Promise<void> {
    if (this.seeded) return;
    await getRepository().seedSuperAdmins(this.superAdminPhones());
    this.seeded = true;
  }

  async listAdmins(): Promise<AdminUser[]> {
    await this.seedIfNeeded();
    return getRepository().listAdmins();
  }

  async loginByPhone(rawPhone: string, options?: { chatId?: number; maxLogin?: string }): Promise<{ token: string; admin: AdminUser } | null> {
    await this.seedIfNeeded();
    return getRepository().loginAdminByPhone(rawPhone, options);
  }

  async getAdminByToken(token: string): Promise<AdminUser | null> {
    await this.seedIfNeeded();
    return getRepository().getAdminByToken(token);
  }

  async createInvite(actor: AdminUser, rawPhone: string, role: AdminRole): Promise<Invite> {
    await this.seedIfNeeded();
    return getRepository().createInvite(actor, rawPhone, role);
  }

  async activateInvite(code: string, rawPhone: string): Promise<AdminUser> {
    await this.seedIfNeeded();
    return getRepository().activateInvite(code, rawPhone);
  }

  async listInvites(): Promise<Invite[]> {
    await this.seedIfNeeded();
    return getRepository().listInvites();
  }

  async updateAdminRole(actor: AdminUser, adminId: number, role: AdminRole): Promise<AdminUser> {
    await this.seedIfNeeded();
    return getRepository().updateAdminRole(actor, adminId, role);
  }

  async setAdminActive(actor: AdminUser, adminId: number, active: boolean): Promise<AdminUser> {
    await this.seedIfNeeded();
    return getRepository().setAdminActive(actor, adminId, active);
  }

  async bindAdminChat(rawPhone: string, chatId: number, maxLogin?: string): Promise<AdminUser | null> {
    await this.seedIfNeeded();
    return getRepository().bindAdminChat(rawPhone, chatId, maxLogin);
  }

  async addFeedEvent(event: Omit<FeedEvent, 'id' | 'created_at'>): Promise<FeedEvent> {
    await this.seedIfNeeded();
    return getRepository().addFeedEvent(event);
  }

  async getFeed(limit = 200): Promise<FeedEvent[]> {
    await this.seedIfNeeded();
    return getRepository().getFeed(limit);
  }
}

export { normalizePhone };
export type { AdminRole, AdminUser, FeedEvent, Invite };
