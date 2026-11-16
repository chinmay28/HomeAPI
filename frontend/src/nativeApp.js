/**
 * The Android app, when this page is running inside it.
 *
 * The app's page bridge (apps/android/app/src/main/assets/homeapi-bridge.js)
 * defines `window.homeapiApp` before any of the page's own scripts run, on the
 * server's origin only. In a browser there is no such object and this returns
 * null, so everything that depends on it is simply not shown.
 *
 * @returns {{platform: string, mode: string, openSettings: () => Promise<unknown>} | null}
 */
export function nativeApp() {
  const app = typeof window === 'undefined' ? undefined : window.homeapiApp;
  return app && typeof app.openSettings === 'function' ? app : null;
}
