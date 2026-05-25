export type BotFlow =
  | { step: 'idle' }
  | { step: 'auth_wait_consent' }
  | { step: 'claim_wait_title' }
  | { step: 'claim_wait_description'; title: string }
  | { step: 'claim_wait_category'; title: string; description: string }
  | { step: 'claim_wait_lat'; title: string; description: string; category: string }
  | { step: 'claim_wait_long'; title: string; description: string; category: string; latitude: number };

export type BotSession = {
  chatId: number;
  maxUserId?: number;
  maxLogin?: string;
  consentAccepted?: boolean;
  consentAcceptedAt?: number;
  consentVersion?: string;
  flow: BotFlow;
  updatedAt: number;
};

export type SessionDb = {
  sessions: Record<string, BotSession>;
};
