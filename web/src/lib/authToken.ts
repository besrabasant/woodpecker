const STORAGE_KEY = 'woodpecker_jwt';

export function isValidJwt(token: string | null): token is string {
  return typeof token === 'string' && token.split('.').length === 3;
}

export function getStoredAuthToken(): string | null {
  try {
    const token = window.localStorage.getItem(STORAGE_KEY);
    if (!isValidJwt(token)) {
      window.localStorage.removeItem(STORAGE_KEY);
      return null;
    }
    return token;
  } catch {
    return null;
  }
}

export function storeAuthToken(token: string | null): void {
  try {
    if (!isValidJwt(token)) {
      window.localStorage.removeItem(STORAGE_KEY);
      return;
    }
    window.localStorage.setItem(STORAGE_KEY, token);
  } catch {
    // ignore storage errors (e.g., disabled cookies)
  }
}
