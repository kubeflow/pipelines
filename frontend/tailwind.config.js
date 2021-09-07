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
              dark: '#087f23',
            },
          },
          yellow: {
            100: {
              DEFAULT: '#fff9c4',
            },
            400: {
              DEFAULT: '#ffee58',
              light: '#ffff8b',
            },
            500: {
              DEFAULT: '#ffeb3b',
              dark: '#c8b900',
            },
            600: {
              DEFAULT: '#fdd835',
              dark: '#c6a700',
            },
          },
          organge: {
            300: {
              DEFAULT: '#ffb74d',
            },
          },
          grey: {
            100: {
              DEFAULT: '#f5f5f5',
            },
            200: {
              DEFAULT: '#eeeeee',
            },
            300: {
              DEFAULT: '#e0e0e0',
              dark: '#aeaeae',
            },
            400: {
              DEFAULT: '#bdbdbd',
            },
            500: {
              DEFAULT: '#9e9e9e',
            },
          },
          red: {
            100: {
              DEFAULT: '#ffcdd2',
            },
            500: {
              DEFAULT: '#f44336',
            },
          },
          blue: {
            100: {
              DEFAULT: '#bbdefb',
              light: '#eeffff',
            },
            500: {
              DEFAULT: '#2196f3',
            },
          },
          lightblue: {
            100: {
              DEFAULT: '#b3e5fc',
            },
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
