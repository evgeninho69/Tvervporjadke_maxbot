import { MaxClient } from '../api/maxClient.js';
import { env } from '../config/env.js';
import { log } from '../utils/log.js';
import { buildReply } from './commandRouter.js';
import { CoreApiClient } from '../api/coreApiClient.js';
import { extractMessage, updateIdFor } from './updateParser.js';

const maxClient = new MaxClient();
const coreApiClient = new CoreApiClient();

export async function startPolling(): Promise<void> {
  let marker: number | undefined;

  log.info('MAX polling worker started', {
    pollIntervalMs: env.MAX_POLL_INTERVAL_MS
  });

  while (true) {
    try {
      const response = await maxClient.getUpdates(marker);
      marker = response.marker ?? marker;

      for (const update of response.updates ?? []) {
        const message = extractMessage(update);
        if (!message) {
          continue;
        }

        const updateId = updateIdFor(update, message);
        if (updateId) {
          const claimed = await coreApiClient.claimIncomingUpdate({
            updateId,
            source: 'polling',
            chatId: message.chat?.chat_id ?? message.recipient?.chat_id,
            messageId: message.message_id || message.body?.mid
          });

          if (!claimed.claimed) {
            log.info('Skip duplicate MAX polling update', { updateId });
            continue;
          }
        }

        log.info('Incoming MAX message', {
          chatId: message.chat?.chat_id,
          text: message.text ?? ''
        });

        try {
          const reply = await buildReply(message);
          if (!reply) {
            continue;
          }

          await maxClient.sendMessage(reply);
          log.info('Outgoing MAX message sent', {
            chatId: reply.chat_id
          });
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
          log.error('Failed to process incoming message', {
            chatId: message.chat?.chat_id,
            error: error instanceof Error ? error.message : 'unknown_error'
          });
        }
      }
    } catch (error) {
      log.error('Polling iteration failed', {
        error: error instanceof Error ? error.message : 'unknown_error'
      });
    }

    await new Promise((resolve) => setTimeout(resolve, env.MAX_POLL_INTERVAL_MS));
  }
}
