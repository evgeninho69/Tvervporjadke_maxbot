import type { Express, NextFunction, Request, Response } from 'express';
import { env } from '../config/env.js';
import { ADMIN_ROLES, CLAIM_STATUSES, parseClaimStatus, statusLabel } from '../domain/models.js';
import { AdminStore, normalizePhone, type AdminRole, type AdminUser } from '../storage/adminStore.js';
import { getRepository } from '../storage/repository.js';
import type { BotSessionRecord, ClaimRecord, UserRecord } from '../storage/types.js';

const adminStore = new AdminStore();
const repository = getRepository();

const CITIZEN_CATEGORIES = [
  { id: 'roads', label: 'Дороги', icon: '🛣️' },
  { id: 'housing', label: 'ЖКХ', icon: '🏠' },
  { id: 'improvement', label: 'Благоустройство', icon: '🌳' },
  { id: 'transport', label: 'Транспорт', icon: '🚌' },
  { id: 'proposal', label: 'Предложения', icon: '💡' }
] as const;

type AsyncHandler = (req: Request, res: Response, next: NextFunction) => Promise<void>;

function wrap(handler: AsyncHandler) {
  return (req: Request, res: Response, next: NextFunction): void => {
    handler(req, res, next).catch(next);
  };
}

function parseChatId(req: Request): number {
  const source = req.query.chat_id ?? req.body?.chat_id;
  const chatId = Number(source);
  return Number.isFinite(chatId) && chatId > 0 ? chatId : 0;
}

function parseMaxUserId(req: Request): number | undefined {
  const source = req.query.max_user_id ?? req.body?.max_user_id;
  const value = Number(source);
  return Number.isFinite(value) && value > 0 ? value : undefined;
}

function parseMaxLogin(req: Request): string | undefined {
  const value = String(req.query.max_login ?? req.body?.max_login ?? '').trim();
  return value || undefined;
}

async function upsertUserFromReq(req: Request): Promise<{ chatId: number; maxUserId?: number; maxLogin?: string }> {
  const chatId = parseChatId(req);
  const maxUserId = parseMaxUserId(req);
  const maxLogin = parseMaxLogin(req);

  if (chatId > 0) {
    await repository.upsertUser({ chatId, maxUserId, maxLogin });
  }

  return { chatId, maxUserId, maxLogin };
}

function bearerToken(req: Request): string {
  const raw = req.headers.authorization || '';
  return raw.toLowerCase().startsWith('bearer ') ? raw.slice(7).trim() : '';
}

function internalToken(req: Request): string {
  const token = req.headers['x-internal-token'];
  if (Array.isArray(token)) return token[0] || '';
  return typeof token === 'string' ? token : '';
}

async function requireAdmin(req: Request, res: Response): Promise<AdminUser | null> {
  const token = bearerToken(req);
  if (!token) {
    res.status(401).json({ error: 'missing bearer token' });
    return null;
  }

  const admin = await adminStore.getAdminByToken(token);
  if (!admin) {
    res.status(401).json({ error: 'invalid session' });
    return null;
  }

  return admin;
}

function requireInternal(req: Request, res: Response): boolean {
  if (internalToken(req) !== env.CORE_API_INTERNAL_TOKEN) {
    res.status(401).json({ error: 'invalid internal token' });
    return false;
  }

  return true;
}

function profileToDto(user: UserRecord | null, identity: { chatId: number; maxUserId?: number; maxLogin?: string }) {
  return {
    chat_id: identity.chatId || undefined,
    max_user_id: user?.max_user_id ?? identity.maxUserId,
    max_login: user?.max_login || identity.maxLogin || '',
    consent_accepted: Boolean(user?.consent_accepted),
    notifications: {
      push_enabled: user?.notification_push ?? true
    }
  };
}

function claimToDto(claim: ClaimRecord) {
  return {
    id: claim.id,
    chat_id: claim.chat_id,
    category: claim.category,
    title: claim.title,
    description: claim.description,
    status: claim.status,
    status_label: statusLabel(claim.status),
    media: claim.photos.slice(0, 3).map((url) => ({ url })),
    result_media: claim.result_media.map((url) => ({ url })),
    location: {
      lat: claim.latitude,
      lng: claim.longitude,
      address_fallback: claim.address_fallback || undefined
    },
    official_response: claim.official_response || claim.feedback || undefined,
    assignee: claim.assignee || undefined,
    created_at: claim.created_at,
    updated_at: claim.updated_at
  };
}

function parseCoordinates(input: unknown): { latitude: number; longitude: number; addressFallback?: string } {
  const body = typeof input === 'object' && input !== null ? (input as Record<string, unknown>) : {};
  const lat = Number(body.lat ?? body.latitude);
  const lng = Number(body.lng ?? body.longitude);
  const addressFallback = String(body.address_fallback ?? '').trim();

  return {
    latitude: Number.isFinite(lat) ? lat : 0,
    longitude: Number.isFinite(lng) ? lng : 0,
    addressFallback: addressFallback || undefined
  };
}

function parseMedia(input: unknown): string[] {
  if (!Array.isArray(input)) return [];
  return input
    .filter((item) => typeof item === 'string')
    .map((item) => item.trim())
    .filter(Boolean)
    .slice(0, 3);
}

type LegacyClaimStatus = 'pending' | 'accepted' | 'completed' | 'declined';

function toLegacyClaimStatus(status: ClaimRecord['status']): LegacyClaimStatus {
  switch (status) {
    case 'moderation':
      return 'pending';
    case 'in_work':
      return 'accepted';
    case 'resolved':
      return 'completed';
    case 'rejected':
      return 'declined';
    default:
      return 'pending';
  }
}

function claimToLegacyV1Dto(claim: ClaimRecord) {
  return {
    id: claim.id,
    chat_id: claim.chat_id,
    type: claim.type,
    title: claim.title || '',
    description: claim.description,
    category: claim.category,
    latitude: claim.latitude,
    longitude: claim.longitude,
    photos: claim.photos,
    status: toLegacyClaimStatus(claim.status),
    feedback: claim.official_response || claim.feedback || undefined,
    created_at: claim.created_at,
    updated_at: claim.updated_at
  };
}

function parseLegacyConsentAccepted(body: unknown): boolean {
  if (!body || typeof body !== 'object') return false;
  const source = body as Record<string, unknown>;
  if (typeof source.accepted === 'boolean') return source.accepted;
  if (typeof source.consent_accepted === 'boolean') return source.consent_accepted;
  return Boolean(source.accepted ?? source.consent_accepted);
}

function parseBotSession(chatId: number, body: Record<string, unknown>): BotSessionRecord {
  const flow = typeof body.flow_json === 'object' && body.flow_json !== null ? (body.flow_json as Record<string, unknown>) : { step: 'idle' };
  const maxUserId = Number(body.max_user_id);
  const consentAt = Number(body.consent_accepted_at);
  const updatedAt = Number(body.updated_at);

  return {
    chat_id: chatId,
    flow_json: flow,
    max_user_id: Number.isFinite(maxUserId) && maxUserId > 0 ? maxUserId : undefined,
    max_login: String(body.max_login || '').trim() || undefined,
    consent_accepted: typeof body.consent_accepted === 'boolean' ? body.consent_accepted : undefined,
    consent_accepted_at: Number.isFinite(consentAt) && consentAt > 0 ? consentAt : undefined,
    consent_version: String(body.consent_version || '').trim() || undefined,
    updated_at: Number.isFinite(updatedAt) && updatedAt > 0 ? updatedAt : Date.now()
  };
}

export function setupApiV2(app: Express): void {
  app.get('/v2/citizen/categories', (_req, res) => {
    res.status(200).json({ categories: CITIZEN_CATEGORIES });
  });

  // Legacy v1 compatibility for old MAX mini app bundles that still call /v1/*.
  app.get('/v1/categories/', (_req, res) => {
    res.status(200).json({
      categories: CITIZEN_CATEGORIES.map((item) => ({
        name: item.label,
        subcategories: [item.label]
      }))
    });
  });

  app.get('/v1/me/', wrap(async (req, res) => {
    const identity = await upsertUserFromReq(req);
    const user = await repository.getUserByChatId(identity.chatId);
    const admins = identity.chatId > 0 ? await adminStore.listAdmins() : [];
    const admin = admins.find((item) => item.max_chat_id === identity.chatId) || null;

    res.status(200).json({
      email: identity.chatId > 0 ? `max-user-${identity.chatId}@tver4ever.ru` : undefined,
      profile: user
        ? {
          chat_id: user.chat_id,
          max_user_id: user.max_user_id,
          max_login: user.max_login || undefined,
          consent_accepted: Boolean(user.consent_accepted),
          consent_version: user.consent_version || undefined
        }
        : null,
      admin: admin
        ? {
          role: admin.role,
          phone: admin.phone
        }
        : null
    });
  }));

  app.get('/v1/consent', wrap(async (req, res) => {
    const identity = await upsertUserFromReq(req);
    const user = await repository.getUserByChatId(identity.chatId);
    res.status(200).json({ accepted: Boolean(user?.consent_accepted) });
  }));

  app.post('/v1/consent', wrap(async (req, res) => {
    const identity = await upsertUserFromReq(req);
    if (identity.chatId <= 0) {
      res.status(400).json({ error: 'chat_id is required' });
      return;
    }

    const accepted = parseLegacyConsentAccepted(req.body);
    await repository.setUserConsent(identity.chatId, accepted, '152fz-v2');
    res.status(200).json({ accepted });
  }));

  app.get('/v1/my/claims/chunk', wrap(async (req, res) => {
    const identity = await upsertUserFromReq(req);
    if (identity.chatId <= 0) {
      res.status(200).json({ claims: [] });
      return;
    }

    const pageSizeRaw = Number(req.query.page_size);
    const pageSize = Number.isFinite(pageSizeRaw) && pageSizeRaw > 0 ? Math.min(pageSizeRaw, 100) : 100;
    const claims = await repository.listClaimsByChat(identity.chatId, pageSize);

    res.status(200).json({ claims: claims.map(claimToLegacyV1Dto) });
  }));

  app.get('/v2/citizen/profile', wrap(async (req, res) => {
    const identity = await upsertUserFromReq(req);
    const user = await repository.getUserByChatId(identity.chatId);
    res.status(200).json({ profile: profileToDto(user, identity) });
  }));

  app.post('/v2/citizen/consent', wrap(async (req, res) => {
    const identity = await upsertUserFromReq(req);
    if (identity.chatId <= 0) {
      res.status(400).json({ error: 'chat_id is required' });
      return;
    }

    const accepted = Boolean(req.body?.accepted);
    await repository.setUserConsent(identity.chatId, accepted, '152fz-v2');
    res.status(200).json({ accepted });
  }));

  app.post('/v2/citizen/notifications', wrap(async (req, res) => {
    const identity = await upsertUserFromReq(req);
    if (identity.chatId <= 0) {
      res.status(400).json({ error: 'chat_id is required' });
      return;
    }

    const pushEnabled = Boolean(req.body?.push_enabled);
    await repository.setNotificationPush(identity.chatId, pushEnabled);
    res.status(200).json({ push_enabled: pushEnabled });
  }));

  app.get('/v2/citizen/announcements', wrap(async (_req, res) => {
    const announcements = await repository.listAnnouncements(30);
    res.status(200).json({ announcements });
  }));

  app.get('/v2/citizen/bootstrap', wrap(async (req, res) => {
    const identity = await upsertUserFromReq(req);
    const user = await repository.getUserByChatId(identity.chatId);
    const claims = identity.chatId > 0 ? await repository.listClaimsByChat(identity.chatId, 20) : [];
    const announcements = await repository.listAnnouncements(20);

    res.status(200).json({
      profile: profileToDto(user, identity),
      categories: CITIZEN_CATEGORIES,
      claims: claims.map(claimToDto),
      announcements
    });
  }));

  app.get('/v2/citizen/claims', wrap(async (req, res) => {
    const identity = await upsertUserFromReq(req);
    const claims = identity.chatId > 0 ? await repository.listClaimsByChat(identity.chatId) : [];
    res.status(200).json({ claims: claims.map(claimToDto) });
  }));

  app.get('/v2/citizen/claims/:id', wrap(async (req, res) => {
    const identity = await upsertUserFromReq(req);
    const claim = await repository.getClaimById(Number(req.params.id));

    if (!claim || claim.chat_id !== identity.chatId) {
      res.status(404).json({ error: 'claim not found' });
      return;
    }

    res.status(200).json({ claim: claimToDto(claim) });
  }));

  app.post('/v2/citizen/claims', wrap(async (req, res) => {
    const identity = await upsertUserFromReq(req);
    if (identity.chatId <= 0) {
      res.status(400).json({ error: 'chat_id is required' });
      return;
    }

    const user = await repository.getUserByChatId(identity.chatId);
    if (!user?.consent_accepted) {
      res.status(403).json({ error: 'consent_required' });
      return;
    }

    const category = String(req.body?.category || '').trim();
    const description = String(req.body?.description || '').trim();
    const media = parseMedia(req.body?.media);
    const location = parseCoordinates(req.body?.location);

    if (!category) {
      res.status(400).json({ error: 'category is required' });
      return;
    }
    if (description.length < 5 || description.length > 1000) {
      res.status(400).json({ error: 'description must be 5..1000 chars' });
      return;
    }

    const hasPoint = location.latitude !== 0 && location.longitude !== 0;
    if (!hasPoint && !location.addressFallback) {
      res.status(400).json({ error: 'location point or address_fallback is required' });
      return;
    }

    const claim = await repository.createClaim({
      chatId: identity.chatId,
      category,
      description,
      latitude: location.latitude,
      longitude: location.longitude,
      addressFallback: location.addressFallback,
      media
    });

    await adminStore.addFeedEvent({
      type: 'claim_created',
      text: `Новая заявка #${claim.id}: ${category}`,
      chat_id: identity.chatId,
      claim_id: claim.id,
      payload: {
        category,
        description,
        latitude: location.latitude,
        longitude: location.longitude,
        address_fallback: location.addressFallback || null
      }
    });

    await repository.createDomainEvent('claim_status_changed', 'claim', claim.id, {
      claim_id: claim.id,
      chat_id: claim.chat_id,
      status: 'moderation',
      status_label: statusLabel('moderation'),
      category,
      description,
      official_response: null
    });

    res.status(201).json({ claim_id: claim.id });
  }));

  app.post('/admin/v2/auth/login', wrap(async (req, res) => {
    const phone = normalizePhone(String(req.body?.phone ?? ''));
    if (!phone) {
      res.status(400).json({ error: 'phone is required' });
      return;
    }

    const session = await adminStore.loginByPhone(phone, {
      chatId: parseChatId(req) || undefined,
      maxLogin: parseMaxLogin(req)
    });

    if (!session) {
      res.status(403).json({ error: 'admin not found or inactive' });
      return;
    }

    res.status(200).json(session);
  }));

  app.post('/admin/v2/auth/invite-activate', wrap(async (req, res) => {
    const admin = await adminStore.activateInvite(String(req.body?.code ?? '').trim(), normalizePhone(String(req.body?.phone ?? '')));
    res.status(200).json({ admin });
  }));

  app.get('/admin/v2/me', wrap(async (req, res) => {
    const admin = await requireAdmin(req, res);
    if (!admin) return;
    res.status(200).json({ admin });
  }));

  app.get('/admin/v2/dashboard', wrap(async (req, res) => {
    const admin = await requireAdmin(req, res);
    if (!admin) return;

    const metrics = await repository.getDashboardMetrics();
    const feed = await adminStore.getFeed(60);

    res.status(200).json({
      role: admin.role,
      ...metrics,
      latest_feed: feed.slice(0, 20)
    });
  }));

  app.get('/admin/v2/queue', wrap(async (req, res) => {
    const admin = await requireAdmin(req, res);
    if (!admin) return;

    const filter = parseClaimStatus(req.query.status);
    const claims = await repository.listClaims(filter || undefined);
    res.status(200).json({ claims: claims.map(claimToDto), available_statuses: CLAIM_STATUSES });
  }));

  app.get('/admin/v2/claims/:id', wrap(async (req, res) => {
    const admin = await requireAdmin(req, res);
    if (!admin) return;

    const claim = await repository.getClaimById(Number(req.params.id));
    if (!claim) {
      res.status(404).json({ error: 'claim not found' });
      return;
    }

    res.status(200).json({ claim: claimToDto(claim) });
  }));

  app.post('/admin/v2/claims/:id/assign', wrap(async (req, res) => {
    const admin = await requireAdmin(req, res);
    if (!admin) return;

    const assignee = String(req.body?.assignee ?? '').trim();
    if (!assignee) {
      res.status(400).json({ error: 'assignee is required' });
      return;
    }

    const claim = await repository.updateClaimAssignment(Number(req.params.id), assignee);
    if (!claim) {
      res.status(404).json({ error: 'claim not found' });
      return;
    }

    res.status(200).json({ claim: claimToDto(claim) });
  }));

  app.post('/admin/v2/claims/:id/status', wrap(async (req, res) => {
    const admin = await requireAdmin(req, res);
    if (!admin) return;

    const nextStatus = parseClaimStatus(req.body?.status);
    if (!nextStatus) {
      res.status(400).json({ error: 'invalid status' });
      return;
    }

    const result = await repository.updateClaimStatus({
      claimId: Number(req.params.id),
      status: nextStatus,
      officialResponse: String(req.body?.official_response ?? '').trim(),
      assignee: String(req.body?.assignee ?? '').trim(),
      resultMedia: parseMedia(req.body?.result_media)
    });

    if (!result) {
      res.status(404).json({ error: 'claim not found' });
      return;
    }

    await adminStore.addFeedEvent({
      type: 'claim_updated',
      text: `Заявка #${result.current.id}: ${statusLabel(result.previous.status)} -> ${statusLabel(result.current.status)}`,
      claim_id: result.current.id,
      payload: {
        previous_status: result.previous.status,
        next_status: result.current.status,
        by_admin: admin.phone,
        official_response: result.current.official_response || null
      }
    });

    if (result.previous.status !== result.current.status) {
      await repository.createDomainEvent('claim_status_changed', 'claim', result.current.id, {
        claim_id: result.current.id,
        chat_id: result.current.chat_id,
        status: result.current.status,
        status_label: statusLabel(result.current.status),
        official_response: result.current.official_response || result.current.feedback || null,
        result_media: result.current.result_media,
        category: result.current.category,
        description: result.current.description
      });
    }

    res.status(200).json({ claim: claimToDto(result.current) });
  }));

  app.get('/admin/v2/messages', wrap(async (req, res) => {
    const admin = await requireAdmin(req, res);
    if (!admin) return;
    res.status(200).json({ messages: await adminStore.getFeed(Math.max(1, Math.min(Number(req.query.limit ?? 200), 2000))) });
  }));

  app.get('/admin/v2/admins', wrap(async (req, res) => {
    const admin = await requireAdmin(req, res);
    if (!admin) return;
    res.status(200).json({ admins: await adminStore.listAdmins() });
  }));

  app.post('/admin/v2/admins/:id/role', wrap(async (req, res) => {
    const actor = await requireAdmin(req, res);
    if (!actor) return;

    const role = String(req.body?.role ?? '').trim() as AdminRole;
    if (!ADMIN_ROLES.includes(role)) {
      res.status(400).json({ error: 'invalid role' });
      return;
    }

    const admin = await adminStore.updateAdminRole(actor, Number(req.params.id), role);
    res.status(200).json({ admin });
  }));

  app.post('/admin/v2/admins/:id/active', wrap(async (req, res) => {
    const actor = await requireAdmin(req, res);
    if (!actor) return;
    const admin = await adminStore.setAdminActive(actor, Number(req.params.id), Boolean(req.body?.active));
    res.status(200).json({ admin });
  }));

  app.post('/admin/v2/invites', wrap(async (req, res) => {
    const actor = await requireAdmin(req, res);
    if (!actor) return;

    const role = String(req.body?.role ?? 'operator').trim() as AdminRole;
    if (!ADMIN_ROLES.includes(role)) {
      res.status(400).json({ error: 'invalid role' });
      return;
    }

    const invite = await adminStore.createInvite(actor, normalizePhone(String(req.body?.phone ?? '')), role);
    res.status(200).json({ invite });
  }));

  app.get('/admin/v2/invites', wrap(async (req, res) => {
    const admin = await requireAdmin(req, res);
    if (!admin) return;
    res.status(200).json({ invites: await adminStore.listInvites() });
  }));

  app.post('/internal/v2/bot/admin-bind', wrap(async (req, res) => {
    if (!requireInternal(req, res)) return;

    const admin = await adminStore.bindAdminChat(
      normalizePhone(String(req.body?.phone ?? '')),
      Number(req.body?.chat_id ?? 0),
      String(req.body?.max_login ?? '').trim() || undefined
    );

    if (!admin) {
      res.status(404).json({ error: 'admin phone not found' });
      return;
    }

    res.status(200).json({ admin });
  }));

  app.post('/internal/v2/bot/feed-event', wrap(async (req, res) => {
    if (!requireInternal(req, res)) return;

    const type = String(req.body?.type ?? '').trim() as 'incoming_message' | 'claim_created' | 'claim_updated' | 'status_notification';
    const text = String(req.body?.text ?? '').trim();
    if (!type || !text) {
      res.status(400).json({ error: 'type and text are required' });
      return;
    }

    const event = await adminStore.addFeedEvent({
      type,
      text,
      chat_id: Number(req.body?.chat_id ?? 0) || undefined,
      claim_id: Number(req.body?.claim_id ?? 0) || undefined,
      payload: typeof req.body?.payload === 'object' && req.body.payload ? (req.body.payload as Record<string, unknown>) : undefined
    });

    res.status(200).json({ event });
  }));

  app.post('/internal/v2/bot/incoming-updates/claim', wrap(async (req, res) => {
    if (!requireInternal(req, res)) return;

    const updateId = String(req.body?.update_id ?? '').trim();
    if (!updateId) {
      res.status(400).json({ error: 'update_id is required' });
      return;
    }

    const claimed = await repository.claimIncomingUpdate(
      updateId,
      String(req.body?.source ?? 'unknown').trim(),
      Number(req.body?.chat_id ?? 0) || undefined,
      String(req.body?.message_id ?? '').trim() || undefined
    );

    res.status(200).json({ claimed });
  }));

  app.post('/internal/v2/bot/incoming-updates/complete', wrap(async (req, res) => {
    if (!requireInternal(req, res)) return;

    const updateId = String(req.body?.update_id ?? '').trim();
    const result = String(req.body?.result ?? '').trim();
    if (!updateId || !['processed', 'failed'].includes(result)) {
      res.status(400).json({ error: 'update_id and result=processed|failed are required' });
      return;
    }

    await repository.completeIncomingUpdate(updateId, result as 'processed' | 'failed', String(req.body?.error ?? '').trim() || undefined);
    res.status(200).json({ ok: true });
  }));

  app.get('/internal/v2/bot/last-claim-status', wrap(async (req, res) => {
    if (!requireInternal(req, res)) return;

    const chatId = Number(req.query.chat_id ?? 0);
    const claimId = Number(req.query.claim_id ?? 0);
    const claim = Number.isFinite(claimId) && claimId > 0
      ? await repository.getClaimById(claimId)
      : (await repository.listClaimsByChat(chatId, 1))[0];

    if (!claim || (Number.isFinite(chatId) && chatId > 0 && claim.chat_id !== chatId)) {
      res.status(404).json({ error: 'claim not found' });
      return;
    }

    res.status(200).json({
      claim: {
        id: claim.id,
        status: claim.status,
        status_label: statusLabel(claim.status),
        category: claim.category,
        updated_at: claim.updated_at,
        official_response: claim.official_response || claim.feedback || null
      }
    });
  }));

  app.get('/internal/v2/bot/sessions/:chatId', wrap(async (req, res) => {
    if (!requireInternal(req, res)) return;
    const session = await repository.getBotSession(Number(req.params.chatId));
    res.status(200).json({ session });
  }));

  app.put('/internal/v2/bot/sessions/:chatId', wrap(async (req, res) => {
    if (!requireInternal(req, res)) return;
    const session = parseBotSession(Number(req.params.chatId), req.body as Record<string, unknown>);
    await repository.saveBotSession(session);
    res.status(200).json({ session });
  }));

  app.delete('/internal/v2/bot/sessions/:chatId', wrap(async (req, res) => {
    if (!requireInternal(req, res)) return;
    const session = await repository.clearBotSession(Number(req.params.chatId), Date.now());
    res.status(200).json({ session });
  }));

  app.get('/internal/v2/events/pending', wrap(async (req, res) => {
    if (!requireInternal(req, res)) return;
    const events = await repository.listPendingDomainEvents(Math.max(1, Math.min(Number(req.query.limit ?? 30), 100)));
    res.status(200).json({ events });
  }));

  app.post('/internal/v2/events/:id/ack', wrap(async (req, res) => {
    if (!requireInternal(req, res)) return;
    const event = await repository.ackDomainEvent(Number(req.params.id));
    if (!event) {
      res.status(404).json({ error: 'event not found' });
      return;
    }
    res.status(200).json({ event });
  }));

  app.post('/internal/v2/events/:id/fail', wrap(async (req, res) => {
    if (!requireInternal(req, res)) return;
    const error = String(req.body?.error ?? 'unknown delivery error').trim() || 'unknown delivery error';
    const event = await repository.failDomainEvent(
      Number(req.params.id),
      error,
      env.DOMAIN_EVENT_MAX_ATTEMPTS,
      env.DOMAIN_EVENT_RETRY_BASE_MS,
      env.DOMAIN_EVENT_RETRY_MAX_MS
    );
    if (!event) {
      res.status(404).json({ error: 'event not found' });
      return;
    }
    res.status(200).json({ event });
  }));

  app.use((error: unknown, _req: Request, res: Response, _next: NextFunction) => {
    const message = error instanceof Error ? error.message : 'internal error';
    res.status(500).json({ error: message });
  });
}
