// Overrides Hextra's assets/js/head/theme.js.
//
// This site is dark-only: the theme switcher is disabled and the light palette
// is not maintained. Hextra's version reads `color-theme` from localStorage in
// preference to the configured default, which would strand anyone who toggled
// to light while the switcher existed — with no switcher left to toggle back.
//
// `setTheme` must stay defined and global: Hextra's js/core/theme.js calls it,
// and dropping it throws "setTheme is not defined" on every page.
//
// Kept in js/head (not js/core) so it runs before first paint and cannot flash.

function setTheme(theme) {
  // Dark-only — the requested theme is deliberately ignored.
  document.documentElement.classList.remove('light');
  document.documentElement.classList.add('dark');
  document.documentElement.style.colorScheme = 'dark';
}

setTheme('dark');

try {
  localStorage.removeItem('color-theme');
} catch (e) {
  /* private mode / storage disabled — nothing to clean up */
}
