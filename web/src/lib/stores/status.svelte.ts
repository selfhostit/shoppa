import { errorMessage } from '../pb';

// What the screens show about the connection: loading, offline, or the last
// thing that went wrong.
class Status {
  loading = $state(true);
  error = $state('');
  /** true while the server can't be reached */
  offline = $state(false);
}

export const status = new Status();

/** Runs a change, showing what went wrong, and undoing the local part if it fails. */
export async function run(fn: () => Promise<unknown>, undo?: () => void) {
  status.error = '';
  try {
    await fn();
  } catch (err) {
    undo?.();
    status.error = errorMessage(err);
  }
}
