export {};

// Minimal typing for the Google Identity Services script (accounts.google.com/gsi/client).
declare global {
  interface Window {
    google?: {
      accounts: {
        id: {
          initialize: (config: { client_id: string; callback: (response: { credential: string }) => void }) => void;
          renderButton: (parent: HTMLElement, options: { theme?: string; size?: string; width?: number; text?: string }) => void;
        };
      };
    };
  }
}
