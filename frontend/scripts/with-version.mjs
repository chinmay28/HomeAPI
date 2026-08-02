#!/usr/bin/env node
/**
 * Run a Create React App command with REACT_APP_VERSION set.
 *
 * CRA inlines `process.env.REACT_APP_*` into the bundle at build time, which is
 * the only channel the web client has for a value that can't exist in source —
 * the patch number is the repository's commit count. This wrapper computes it
 * from ../scripts/version.mjs (the same file the Go binary is stamped from, so
 * the header and /api/health always agree) and hands it to the child process.
 *
 * It shells out to node rather than importing so this stays runnable from the
 * frontend workspace regardless of how the root script is resolved, and it is
 * a node script rather than an inline `VAR=$(...)` npm script so `npm start`
 * and `npm run build` work on Windows too.
 *
 * Usage: node scripts/with-version.mjs react-scripts build
 */
import { execFileSync, spawn } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const [command, ...args] = process.argv.slice(2);
if (!command) {
  console.error('usage: node scripts/with-version.mjs <command> [args...]');
  process.exit(2);
}

/** The version string, or null when it can't be computed (no node repo, no git). */
function appVersion() {
  const script = fileURLToPath(new URL('../../scripts/version.mjs', import.meta.url));
  try {
    return execFileSync(process.execPath, [script], { encoding: 'utf8' }).trim();
  } catch (err) {
    // A missing version is not worth failing a build over — the client falls
    // back to its own "unstamped" marker (see src/version.js).
    console.warn(`could not determine app version: ${err.message}`);
    return null;
  }
}

const version = appVersion();
const env = { ...process.env };
if (version) env.REACT_APP_VERSION = version;

const child = spawn(command, args, { stdio: 'inherit', env, shell: process.platform === 'win32' });
child.on('exit', (code, signal) => process.exit(signal ? 1 : (code ?? 1)));
