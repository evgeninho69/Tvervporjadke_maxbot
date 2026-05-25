import { env } from '../config/env.js';
import type { MaxUpdate } from '../types/max.js';
import { log } from '../utils/log.js';
import { buildReply } from './commandRouter.js';
import { MaxClient } from '../api/maxClient.js';
import { CoreApiClient } from '../api/coreApiClient.js';
import { callbackIdFor, extractMessage, updateIdFor } from './updateParser.js';

const maxClient = new MaxClient();
const coreApiClient = new CoreApiClient();

export async function handleWebhook(headers: Record<string, string | string[] | undefined>, body: MaxUpdate): Promise<void> {
  const secret = headers['x-max-bot-api-secret'];
  const secretValue = Array.isArray(secret) ? secret[0] : secret;

  if (!secretValue || secretValue !== env.MAX_WEBHOOK_SECRET) {
    throw new Error('Unauthorized webhook secret');
  }

  if (!['message_created', 'message_callback', 'bot_started'].includes(body.update_type || '')) {
    log.info('Skip unsupported update', { updateType: body.update_type });
    return;
  }

  const message = extractMessage(body);
  if (!message) {
    log.warn('Webhook update has no chat id', { updateType: body.update_type });
    return;
  }

  const updateId = updateIdFor(body, message);
  if (updateId) {
    const claimed = await coreApiClient.claimIncomingUpdate({
      updateId,
      source: 'webhook',
      chatId: message.recipient?.chat_id ?? message.chat?.chat_id,
      messageId: message.message_id || message.body?.mid
    });

    if (!claimed.claimed) {
      log.info('Skip duplicate MAX webhook update', { updateId });
      return;
    }
  }

  const reply = await buildReply(message);
  const callbackId = callbackIdFor(body);

  try {
    if (reply) {
      await maxClient.sendMessage(reply);
    }
    if (callbackId) {
      await maxClient.answerCallback(callbackId);
    }
    if (updateId) {
      await coreApiClient.completeIncomingUpdate({ updateId, result: 'processed' });
    }
  } catch (error) {
    if (updateId) {
      await coreApiClient.completeIncomingUpdate({
        updateId,
        result: 'failed',
        error: error instanceof Error ? error.message : 'unknown_error'
      });
    }
    throw error;
  }
}
