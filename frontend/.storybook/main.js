const path = require('path');

module.exports = {
  // Read https://github.com/storybookjs/storybook/issues/9514
  // for the storybook's maintainer's recommendation for folder location.
  stories: ['../src/**/*.stories.mdx', '../src/**/*.stories.@(js|jsx|ts|tsx)'],
  addons: ['@storybook/addon-links', '@storybook/addon-essentials'],
  core: {
    builder: 'webpack5',
  },
  // https://github.com/storybookjs/presets/issues/97
  webpackFinal: async (config) => {
    config.resolve = config.resolve || {};
    config.resolve.extensions = config.resolve.extensions || ['.js', '.jsx', '.ts', '.tsx'];
    config.resolve.alias = {
      ...(config.resolve.alias || {}),
      src: path.resolve(__dirname, '../src'),
    };

    config.module.rules.push(
      {
        test: /\.(ts|tsx)$/,
        use: [
          {
            loader: require.resolve('babel-loader'),
            options: {
              presets: [
                require.resolve('@babel/preset-env'),
                require.resolve('@babel/preset-react'),
                require.resolve('@babel/preset-typescript'),
              ],
            },
          },
        ],
      },
      {
        test: /\.(png|jpe?g|gif|svg|eot|ttf|woff2?)$/,
        use: [
          {
            loader: require.resolve('file-loader'),
            options: {
              name: 'static/media/[name].[hash:8].[ext]',
            },
          },
        ],
      },
    );

    const oneOfRule = config.module.rules.find((rule) => rule.oneOf);
    if (oneOfRule) {
      const fileLoader = oneOfRule.oneOf.find(
        (rule) => rule.loader && rule.loader.includes('file-loader'),
      );
      if (fileLoader && fileLoader.options && fileLoader.options.name) {
        fileLoader.options.name = `/${fileLoader.options.name}`;
      }
    }

    return config;
  },
};
