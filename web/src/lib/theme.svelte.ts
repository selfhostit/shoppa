// Light or dark, or like the rest of the phone (the default). The choice is
// kept on this phone only: a shared tablet in the kitchen can be dark while a
// phone stays light. app.css's dark variant follows the phone unless <html>
// has the class light or dark.

export type Theme = 'system' | 'light' | 'dark';

const key = 'shoppa:theme';
// the page background in each, for the phone's status bar
const bar = { light: '#fafafa', dark: '#09090b' };

function stored(): Theme {
  try {
    const t = localStorage.getItem(key);
    return t === 'light' || t === 'dark' ? t : 'system';
  } catch {
    return 'system';
  }
}

/** Puts a theme on the page. */
export function apply(theme: Theme, root = document.documentElement) {
  root.classList.toggle('light', theme === 'light');
  root.classList.toggle('dark', theme === 'dark');
  root.style.colorScheme = theme === 'system' ? '' : theme;
  for (const meta of document.querySelectorAll<HTMLMetaElement>('meta[name="theme-color"]')) {
    const own = meta.media.includes('dark') ? bar.dark : bar.light;
    meta.content = theme === 'system' ? own : bar[theme];
  }
}

class ThemeChoice {
  current = $state<Theme>(stored());

  set(theme: Theme) {
    this.current = theme;
    try {
      if (theme === 'system') localStorage.removeItem(key);
      else localStorage.setItem(key, theme);
    } catch {
      // private browsing: it lasts until the page closes
    }
    apply(theme);
  }
}

export const theme = new ThemeChoice();
