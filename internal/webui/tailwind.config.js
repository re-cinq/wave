/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./templates/**/*.html",
  ],
  theme: {
    extend: {
      colors: {
        wave: {
          primary: 'var(--wave-primary)',
          'primary-dark': 'var(--wave-primary-dark)',
          accent: 'var(--wave-accent)',
          secondary: 'var(--wave-secondary)',
          green: 'var(--wave-trust-green)',
          blue: 'var(--wave-trust-blue)',
          warning: 'var(--wave-warning)',
          danger: 'var(--wave-danger)',
        },
        surface: {
          DEFAULT: 'var(--color-bg)',
          secondary: 'var(--color-bg-secondary)',
          tertiary: 'var(--color-bg-tertiary)',
        },
        edge: {
          DEFAULT: 'var(--color-border)',
          light: 'var(--color-border-light)',
        },
        txt: {
          DEFAULT: 'var(--color-text)',
          secondary: 'var(--color-text-secondary)',
          muted: 'var(--color-text-muted)',
        },
        state: {
          completed: 'var(--color-completed)',
          'completed-bg': 'var(--color-completed-bg)',
          running: 'var(--color-running)',
          'running-bg': 'var(--color-running-bg)',
          failed: 'var(--color-failed)',
          'failed-bg': 'var(--color-failed-bg)',
          cancelled: 'var(--color-cancelled)',
          'cancelled-bg': 'var(--color-cancelled-bg)',
          pending: 'var(--color-pending)',
          'pending-bg': 'var(--color-pending-bg)',
        },
      },
      fontFamily: {
        sans: ['var(--font-sans)'],
        mono: ['var(--font-mono)'],
      },
    },
  },
  plugins: [],
};
