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
  ]
}
