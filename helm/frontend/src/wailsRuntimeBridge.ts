// Mirrors the relevant slice of the generated `wailsjs/runtime/runtime.js`
// bridge by hand. The plugin builds to a standalone ES module and cannot
// resolve the main app's `@wailsjs` alias, but `window.runtime` is injected
// by Wails into the shared webview regardless of which bundle calls it —
// see wailsBridge.ts for the fuller explanation.
declare global {
  interface Window {
    runtime: {
      EventsOnMultiple: (
        eventName: string,
        callback: (...data: unknown[]) => void,
        maxCallbacks: number
      ) => () => void;
    };
  }
}

export function EventsOn(eventName: string, callback: (...data: unknown[]) => void): () => void {
  return window.runtime.EventsOnMultiple(eventName, callback, -1);
}
