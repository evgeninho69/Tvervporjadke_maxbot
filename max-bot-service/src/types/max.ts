export type MaxUser = {
  user_id?: number;
  first_name?: string;
  last_name?: string;
  username?: string;
  name?: string;
};

export type MaxChat = {
  chat_id: number;
  title?: string;
};

export type MaxMessage = {
  message_id?: string;
  recipient?: {
    chat_id?: number;
    chat_type?: string;
    user_id?: number;
  };
  body?: {
    text?: string;
    mid?: string;
    seq?: number;
  };
  text?: string;
  chat?: MaxChat;
  from?: MaxUser;
  sender?: MaxUser;
};

export type MaxCallback = {
  callback_id?: string;
  payload?: unknown;
};

export type MaxUpdate = {
  update_id?: string;
  update_type?: string;
  message?: MaxMessage;
  callback?: MaxCallback;
  chat_id?: number;
  user?: MaxUser;
  payload?: unknown;
  timestamp?: number;
};

export type MaxSendMessagePayload = {
  chat_id: number;
  text: string;
  attachments?: Array<{
    type: 'inline_keyboard';
    payload: {
      buttons: Array<Array<{
        type: 'message' | 'link' | 'callback' | 'open_app';
        text: string;
        payload?: string;
        url?: string;
        web_app?: string;
      }>>;
    };
  }>;
};

export type MaxUpdatesResponse = {
  updates: MaxUpdate[];
  marker?: number;
};
