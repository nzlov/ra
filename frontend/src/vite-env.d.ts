/// <reference types="svelte" />
/// <reference types="vite/client" />

interface Window {
  ra?: {
    invoke(action: {type: string; text?: string}): Promise<unknown>;
  };
}
