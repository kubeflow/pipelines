// Compile CSS using the following command:
// npm run build:tailwind
module.exports = {
  content: ['./src/**/*.{js,jsx,ts,tsx}', './public/index.html'],
  darkMode: 'media', // or 'media' or 'class'
  theme: {
    extend: {
      spacing: {
        '112': '28rem',
        '136': '34rem',
      },
      // https://tailwindcss.com/docs/customizing-colors
      colors: {
        // https://material.io/resources/color
        mui: {
          green: {
            50: {
              DEFAULT: '#e8f5e9',
            },
            200: {
              DEFAULT: '#a5d6a7',
              light: '#d7ffd9',
            },
            500: {
              DEFAULT: '#4caf50',
              dark: '#087f23',
            },
            600: {
              DEFAULT: '#43a047',
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
            800: {
              DEFAULT: '#f9a825',
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
            600: {
              DEFAULT: '#757575',
            },
          },
          red: {
            50: {
              DEFAULT: '#ffebee'
            },
            100: {
              DEFAULT: '#ffcdd2',
            },
            500: {
              DEFAULT: '#f44336',
            },
            600: {
              DEFAULT: '#e53935',
            }
          },
          blue: {
            50: {
              DEFAULT: '#e3f2fd'
            },
            100: {
              DEFAULT: '#bbdefb',
              light: '#eeffff',
            },
            500: {
              DEFAULT: '#2196f3',
            },
            600: {
              DEFAULT: '#1e88e5',
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
    extend: {
      borderColor: ['group-focus'],
      textColor: ['group-focus'],
    },
  },
  plugins: [],
};
