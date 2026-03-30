/** @type {import('tailwindcss').Config} */
module.exports = {
  theme: {
    extend: {
      colors: {
        recinq: {
          'midnight-navy': '#0F1F49',
          'crystal-white': '#FFFFFF',
          'quantum-blue': '#0014EB',
          'pulse-blue': '#5664F4',
          'soft-indigo': '#8F96F6',
          'nebula-light': '#E6E8FD',
          'neutral-fog': '#F2F2F7',
        },
      },
      fontFamily: {
        'recinq': ['"Neue Montreal"', '"Helvetica Neue"', 'Helvetica', 'Arial', 'sans-serif'],
        'recinq-logo': ['"Cold Warm"', 'sans-serif'],
      },
      fontSize: {
        'recinq-h1': ['64px', { lineHeight: '1.1', letterSpacing: '-0.02em', fontWeight: '700' }],
        'recinq-h2': ['40px', { lineHeight: '1.2', letterSpacing: '-0.01em', fontWeight: '500' }],
        'recinq-h3': ['24px', { lineHeight: '1.3', letterSpacing: '0', fontWeight: '500' }],
        'recinq-body': ['16px', { lineHeight: '1.5', fontWeight: '400' }],
        'recinq-caption': ['13px', { lineHeight: '1.4', fontWeight: '300' }],
        'recinq-doc-h1': ['22pt', { lineHeight: '1.1', fontWeight: '700' }],
        'recinq-doc-h2': ['16pt', { lineHeight: '1.2', fontWeight: '500' }],
        'recinq-doc-body': ['11pt', { lineHeight: '1.5', fontWeight: '400' }],
      },
      backgroundImage: {
        'recinq-aurora': 'linear-gradient(135deg, #E4E6FD 0%, #5664F4 50%, #8F96F6 100%)',
        'recinq-aurora-dark': 'linear-gradient(135deg, #5664F4 0%, #0F1F49 100%)',
      },
      borderRadius: {
        'recinq-callout': '8px',
      },
    },
  },
};
