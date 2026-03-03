import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { StorybookConfig } from '@storybook/react-vite';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const config: StorybookConfig = {
  stories: ['../src/**/*.stories.@(js|jsx|ts|tsx)'],
  addons: ['@storybook/addon-links'],
  framework: {
    name: '@storybook/react-vite',
    options: {},
  },
  staticDirs: ['../public'],
  viteFinal: async (viteConfig) => {
    viteConfig.resolve = viteConfig.resolve || {};
    const srcAliasPath = path.resolve(__dirname, '../src');
    const existingAlias = viteConfig.resolve.alias;
    if (Array.isArray(existingAlias)) {
      existingAlias.push({ find: 'src', replacement: srcAliasPath });
    } else {
      viteConfig.resolve.alias = {
        ...(existingAlias || {}),
        src: srcAliasPath,
      };
    }
    viteConfig.build = viteConfig.build || {};
    viteConfig.build.commonjsOptions = {
      ...viteConfig.build.commonjsOptions,
      include: [/node_modules/, /src\/generated/, /src\/third_party\/mlmd\/generated/],
    };
    return viteConfig;
  },
};

export default config;
