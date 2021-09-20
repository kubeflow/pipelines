module.exports = {
  // Read https://github.com/storybookjs/storybook/issues/9514 
  // for the storybook's maintainer's recommendation for folder location.
  "stories": [
    "../src/**/*.stories.mdx",
    "../src/**/*.stories.@(js|jsx|ts|tsx)"
  ],
  "addons": [
    "@storybook/addon-links",
    "@storybook/addon-essentials",
    "@storybook/preset-create-react-app"
  ],
  // https://github.com/storybookjs/presets/issues/97
  webpackFinal:  async (config) => {
    const getFileLoader = (config) => {
      const { oneOf = [] } = config.module.rules.find(({ oneOf }) => oneOf);
      return oneOf.find(({loader}) => /file-loader/.test(loader));
    };
    
    const mutateLoaderToMakePathAbsolute = (loader) => {
        if(loader && loader.options && loader.options.name) {
          loader.options.name = '/' + loader.options.name;
        }
    };

    const fileLoader = getFileLoader(config);
    mutateLoaderToMakePathAbsolute(fileLoader);
    return config;
  },
}
