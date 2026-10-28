// Compile CSS using the following command:
// npm run build:tailwind
module.exports = {
  content: ['./index.html', './src/**/*.{js,jsx,ts,tsx}'],
  darkMode: 'media', // or 'media' or 'class'
  theme: {
    extend: {
      spacing: {
        112: '28rem',
        136: '34rem',
      },
    },
  },
  variants: {
    extend: {
      borderColor: ['group-focus'],
      textColor: ['group-focus'],
    },
  },
  plugins: [],
};
