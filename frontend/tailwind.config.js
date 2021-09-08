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
              dark: '#087f23'
            }
          },
          yellow: {
            400: {
              DEFAULT: '#ffee58',
              light: '#ffff8b'
            },
            600: {
              DEFAULT: '#fdd835',
              dark: '#c6a700'
            }
          }
        },
      },
    },
  },
  variants: {
    extend: {},
  },
  plugins: [],
};
