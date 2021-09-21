// Compile CSS using the following command:
// npm run build:tailwind
module.exports = {
  purge: ['./src/**/*.{js,jsx,ts,tsx}', './public/index.html'],
  darkMode: false, // or 'media' or 'class'
  theme: {
    extend: {
      // https://tailwindcss.com/docs/customizing-colors
      colors: {
        // https://material.io/resources/color
        mui: {
          green: {
            200: {
              DEFAULT: '#a5d6a7',
              light: '#d7ffd9',
            },
            500: {
              DEFAULT: '#4caf50',
              strong: '#087f23'
            }
          },
        },
      },
    },
  },
  variants: {
    extend: {},
  },
  plugins: [],
};
