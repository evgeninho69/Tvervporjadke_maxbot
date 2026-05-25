import { env } from '../config/env.js';
import type { MaxMessage, MaxSendMessagePayload, MaxUser } from '../types/max.js';
import type { BotSession } from '../bot/sessionTypes.js';
import { SessionStore } from '../storage/sessionStore.js';
import { CoreApiClient } from '../api/coreApiClient.js';

const sessionStore = new SessionStore();
const coreApiClient = new CoreApiClient();

const CONSENT_VERSION = '152fz-v2';
const START_ONBOARDING_TRIGGER = '🚀 начать';
const ACCEPT_PD_TRIGGER = '✅ принимаю пдн';
const DECLINE_PD_TRIGGER = '❌ не принимаю пдн';
const OPEN_APP_TRIGGER = '📱 открыть приложение';

type KeyboardButton = {
  type: 'message' | 'link' | 'callback' | 'open_app';
  text: string;
  payload?: string;
  url?: string;
  web_app?: string;
};

const START_TEXT = [
  `Здравствуйте. Я официальный бот сервиса «${env.DEFAULT_CHANNEL_NAME}».`,
  'Помогаю отправлять обращения, проверять статусы и связываться с поддержкой.',
  '',
  'Выберите действие кнопками ниже.'
].join('\n');

const WELCOME_TEXT = [
  `Здравствуйте. Я официальный бот сервиса «${env.DEFAULT_CHANNEL_NAME}».`,
  'Перед началом работы подтвердите согласие на обработку персональных данных.',
  '',
  'Нажмите кнопку «Начать».'
].join('\n');

const HELP_TEXT = [
  'Команды бота:',
  '/start - главное меню',
  '/status [id] - проверка статуса обращения',
  '/support - связь с поддержкой',
  '/create - открыть форму обращения в mini app',
  '/myclaims - открыть список ваших заявок',
  '/news - ссылка на канал',
  '/auth - показать согласие 152-ФЗ',
  '/logout - очистить локальную сессию бота'
].join('\n');

const SUPPORT_TEXT = [
  'Поддержка сервиса «Тверь в порядке»',
  '1. Техническая поддержка: support@tver4ever.ru',
  '2. Оператор call-центра: +7 (4822) 00-00-00',
  '3. Канал с новостями: https://max.ru/' + env.DEFAULT_CHANNEL_HANDLE.replace('@', ''),
  '',
  'Если вопрос по заявке, приложите номер обращения в формате: /status 1234'
].join('\n');

const POLICY_TEXT = [
  'Согласие на обработку персональных данных (152-ФЗ):',
  '1. Данные используются для обработки обращений и уведомлений.',
  '2. Для продолжения нажмите «Принимаю ПДн».',
  '3. Отозвать согласие можно через /logout и обращение оператору.'
].join('\n');

function normalizePhone(value: string): string {
  const digits = value.replace(/\D+/g, '');
  if (digits.length === 11 && digits.startsWith('8')) return `+7${digits.slice(1)}`;
  if (digits.length === 11 && digits.startsWith('7')) return `+${digits}`;
  if (digits.length === 10) return `+7${digits}`;
  return value.trim();
}

function getChatId(message: MaxMessage): number | null {
  return message.chat?.chat_id ?? message.recipient?.chat_id ?? null;
}

function getText(message: MaxMessage): string {
  return (message.text ?? message.body?.text ?? '').trim();
}

function getSender(message: MaxMessage): MaxUser | undefined {
  return message.from ?? message.sender;
}

function msg(chatId: number, text: string): MaxSendMessagePayload {
  return { chat_id: chatId, text };
}

function withKeyboard(chatId: number, text: string, rows: KeyboardButton[][]): MaxSendMessagePayload {
  return {
    chat_id: chatId,
    text,
    attachments: [
      {
        type: 'inline_keyboard',
        payload: {
          buttons: rows
        }
      }
    ]
  };
}

async function saveIdle(session: BotSession): Promise<void> {
  session.flow = { step: 'idle' };
  await sessionStore.save(session);
}

function command(text: string): string {
  return text.toLowerCase();
}

function isStartCommand(normalized: string): boolean {
  return normalized === '/start' || normalized === START_ONBOARDING_TRIGGER || normalized === 'начать';
}

function buildMiniAppUrl(session: BotSession, chatId: number, view?: 'home' | 'history' | 'form' | 'profile'): string {
  const url = new URL(env.MAX_MINIAPP_URL);
  url.searchParams.set('chat_id', String(chatId));

  if (session.maxUserId) {
    url.searchParams.set('max_user_id', String(session.maxUserId));
  }

  if (view) {
    url.searchParams.set('view', view);
  }

  return url.toString();
}

function menu(session: BotSession, chatId: number, text: string): MaxSendMessagePayload {
  return withKeyboard(chatId, text, [
    [{ type: 'link', text: '📱 Перейти в приложение', url: buildMiniAppUrl(session, chatId, 'home') }],
    [{ type: 'link', text: '📂 Мои заявки', url: buildMiniAppUrl(session, chatId, 'history') }, { type: 'message', text: '/status' }],
    [{ type: 'message', text: '/support' }, { type: 'message', text: '/help' }]
  ]);
}

function onboardingStart(chatId: number): MaxSendMessagePayload {
  return withKeyboard(chatId, WELCOME_TEXT, [[{ type: 'message', text: '🚀 Начать' }]]);
}

function consentMenu(chatId: number): MaxSendMessagePayload {
  return withKeyboard(chatId, POLICY_TEXT, [[{ type: 'message', text: '✅ Принимаю ПДн' }, { type: 'message', text: '❌ Не принимаю ПДн' }]]);
}

function appLinkMenu(session: BotSession, chatId: number, view: 'home' | 'history' | 'form' | 'profile', text: string): MaxSendMessagePayload {
  return withKeyboard(chatId, text, [[{ type: 'link', text: 'Перейти в приложение', url: buildMiniAppUrl(session, chatId, view) }]]);
}

function channelUrl(): string {
  return `https://max.ru/${env.DEFAULT_CHANNEL_HANDLE.replace('@', '')}`;
}

function newsMenu(chatId: number): MaxSendMessagePayload {
  return withKeyboard(chatId, `Новости проекта «${env.DEFAULT_CHANNEL_NAME}»`, [[{ type: 'link', text: 'Открыть канал', url: channelUrl() }]]);
}

function supportMenu(chatId: number): MaxSendMessagePayload {
  return withKeyboard(chatId, SUPPORT_TEXT, [[{ type: 'link', text: 'Канал проекта', url: channelUrl() }]]);
}

function accountSummary(session: BotSession): string {
  const maxAccount = session.maxLogin ? `MAX: ${session.maxLogin}` : 'MAX: не авторизован';
  const consent = session.consentAccepted ? 'Согласие: принято' : 'Согласие: не принято';
  return `${maxAccount}\n${consent}`;
}

async function processStatusCommand(chatId: number, text: string): Promise<MaxSendMessagePayload> {
  const parts = text.split(/\s+/);
  const maybeId = Number(parts[1]);

  try {
    const data = await coreApiClient.getLastClaimStatus(chatId, Number.isFinite(maybeId) && maybeId > 0 ? maybeId : undefined);
    const claim = data.claim;

    const lines = [
      `Заявка №${claim.id}`,
      `Статус: ${claim.status_label}`,
      claim.category ? `Категория: ${claim.category}` : '',
      claim.updated_at ? `Обновлено: ${claim.updated_at}` : '',
      claim.official_response ? `Ответ: ${claim.official_response}` : ''
    ].filter(Boolean);

    return msg(chatId, lines.join('\n'));
  } catch {
    return msg(chatId, 'Не удалось найти заявку. Проверьте номер или отправьте /create для нового обращения.');
  }
}

export async function buildReply(message: MaxMessage): Promise<MaxSendMessagePayload | null> {
  const chatId = getChatId(message);
  if (!chatId) return null;

  const text = getText(message);
  const normalized = command(text);
  const sender = getSender(message);

  const session = await sessionStore.get(chatId);

  if (sender?.user_id) {
    session.maxUserId = sender.user_id;
    session.maxLogin = sender.username || sender.name || sender.first_name || `id:${sender.user_id}`;
    await sessionStore.save(session);
  }

  void coreApiClient.addFeedEvent({
    type: 'incoming_message',
    text: text || '(empty)',
    chatId,
    payload: {
      max_login: session.maxLogin,
      sender_user_id: sender?.user_id
    }
  }).catch(() => {
    // non-blocking
  });

  if (normalized === '/cancel') {
    await saveIdle(session);
    return msg(chatId, 'Текущий сценарий отменён.');
  }

  if (normalized === '/policy') {
    return msg(chatId, POLICY_TEXT);
  }

  if (normalized === '/accept' || normalized === ACCEPT_PD_TRIGGER) {
    session.consentAccepted = true;
    session.consentAcceptedAt = Date.now();
    session.consentVersion = CONSENT_VERSION;
    session.flow = { step: 'idle' };
    await sessionStore.save(session);
    return menu(session, chatId, `Согласие принято.\n${accountSummary(session)}\n\nТеперь можно работать с обращениями.`);
  }

  if (normalized === '/decline' || normalized === DECLINE_PD_TRIGGER) {
    session.consentAccepted = false;
    session.flow = { step: 'idle' };
    await sessionStore.save(session);
    return onboardingStart(chatId);
  }

  if (isStartCommand(normalized)) {
    if (!session.consentAccepted) {
      if (session.flow.step === 'auth_wait_consent') {
        return consentMenu(chatId);
      }
      session.flow = { step: 'auth_wait_consent' };
      await sessionStore.save(session);
      return onboardingStart(chatId);
    }
    await saveIdle(session);
    return menu(session, chatId, START_TEXT);
  }

  if (normalized === '/help' || normalized === '❓ помощь') {
    return menu(session, chatId, HELP_TEXT);
  }

  if (normalized === '/support') {
    return supportMenu(chatId);
  }

  if (normalized === '/auth') {
    session.flow = { step: 'auth_wait_consent' };
    await sessionStore.save(session);
    return consentMenu(chatId);
  }

  if (normalized === '/logout') {
    const cleared = await sessionStore.clear(chatId);
    return onboardingStart(cleared.chatId);
  }

  if (normalized === '/news' || normalized === '/announcements' || normalized === 'новости' || normalized === '📰 новости') {
    return newsMenu(chatId);
  }

  if (normalized === '/status') {
    return processStatusCommand(chatId, text);
  }

  if (normalized.startsWith('/status ')) {
    return processStatusCommand(chatId, text);
  }

  if (normalized === '/create' || normalized === OPEN_APP_TRIGGER || normalized === 'создать' || normalized === 'создать заявку') {
    if (!session.consentAccepted) {
      return consentMenu(chatId);
    }
    return appLinkMenu(session, chatId, 'form', 'Откройте форму обращения кнопкой ниже.');
  }

  if (normalized === '/myclaims' || normalized === '📂 мои заявки' || normalized === 'мои заявки') {
    if (!session.consentAccepted) {
      return consentMenu(chatId);
    }
    return appLinkMenu(session, chatId, 'history', 'Откройте список заявок кнопкой ниже.');
  }

  if (['/miniapp', 'mini app', 'miniapp'].includes(normalized)) {
    if (!session.consentAccepted) {
      return consentMenu(chatId);
    }
    return appLinkMenu(session, chatId, 'home', 'Откройте приложение кнопкой ниже.');
  }

  if (normalized === '/settings') {
    if (!session.consentAccepted) {
      return consentMenu(chatId);
    }
    return appLinkMenu(session, chatId, 'profile', 'Откройте настройки профиля кнопкой ниже.');
  }

  if (normalized === '/admin') {
    return withKeyboard(chatId, 'Откройте админ-панель кнопкой ниже.', [[{ type: 'link', text: 'Перейти в админ-панель', url: `${env.MAX_MINIAPP_URL}admin` }]]);
  }

  if (normalized.startsWith('/adminbind')) {
    const phoneRaw = text.split(/\s+/).slice(1).join(' ').trim();
    if (!phoneRaw) {
      return msg(chatId, 'Формат: /adminbind +79101234567');
    }

    const phone = normalizePhone(phoneRaw);
    try {
      const bound = await coreApiClient.bindAdminChat(phone, chatId, session.maxLogin);
      return msg(chatId, `Привязка выполнена.\nРоль: ${bound.admin.role}\nТелефон: ${bound.admin.phone}`);
    } catch {
      return msg(chatId, 'Телефон не найден среди администраторов.');
    }
  }

  if (session.flow.step === 'auth_wait_consent') {
    return consentMenu(chatId);
  }

  if (normalized.includes('статус')) {
    return msg(chatId, 'Проверить статус: /status или /status 1234');
  }

  if (normalized.includes('поддерж')) {
    return supportMenu(chatId);
  }

  return msg(chatId, 'Команда не распознана. Отправьте /start для меню.');
}
