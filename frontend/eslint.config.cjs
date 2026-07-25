const eslint = require('@eslint/js');
const typescriptEslint = require('@typescript-eslint/eslint-plugin');
const typescriptParser = require('@typescript-eslint/parser');
const globals = require('globals');
const { createNodeResolver, importX } = require('eslint-plugin-import-x');
const jsxA11y = require('eslint-plugin-jsx-a11y');
const react = require('eslint-plugin-react');
const reactHooks = require('eslint-plugin-react-hooks');

const sourceFiles = ['src/**/*.{js,ts,tsx}', 'server/**/*.ts'];

module.exports = [
  {
    ignores: [
      '**/node_modules/**',
      'src/apis/**',
      'src/apisv2beta1/**',
      'server/dist/**',
      'server/src/generated/**',
      '**/*.test.ts',
      '**/*.test.tsx',
      'src/generated/**',
      'src/third_party/mlmd/generated/**',
      'src/stories/**',
    ],
  },
  {
    files: sourceFiles,
    linterOptions: {
      reportUnusedDisableDirectives: false,
    },
    languageOptions: {
      ecmaVersion: 2020,
      globals: {
        ...globals.browser,
        ...globals.es2020,
        ...globals.node,
      },
      parser: typescriptParser,
      parserOptions: {
        ecmaFeatures: {
          jsx: true,
        },
        sourceType: 'module',
      },
    },
    plugins: {
      '@typescript-eslint': typescriptEslint,
      'import-x': importX,
      'jsx-a11y': jsxA11y,
      react,
      'react-hooks': reactHooks,
    },
    settings: {
      react: {
        version: 'detect',
      },
      'import-x/extensions': ['.ts', '.tsx', '.js', '.jsx'],
      'import-x/external-module-folders': ['node_modules', 'node_modules/@types'],
      'import-x/parsers': {
        '@typescript-eslint/parser': ['.ts', '.tsx'],
      },
      'import-x/resolver-next': [
        createNodeResolver({
          extensions: ['.js', '.jsx', '.ts', '.tsx'],
          modules: [__dirname, 'node_modules'],
        }),
      ],
    },
    rules: {
      ...eslint.configs.recommended.rules,
      ...importX.flatConfigs.recommended.rules,
      ...importX.flatConfigs.typescript.rules,
      ...reactHooks.configs.recommended.rules,
      'no-undef': 'off',
      'no-unused-vars': 'off',
      'no-redeclare': 'off',
      '@typescript-eslint/no-redeclare': 'error',
      '@typescript-eslint/no-unused-vars': [
        'error',
        {
          argsIgnorePattern: '^_',
          varsIgnorePattern: '^_',
          caughtErrors: 'none',
          ignoreRestSiblings: true,
        },
      ],
      'import-x/no-unresolved': [
        'error',
        {
          ignore: ['^src/build/tailwind.output.css$'],
        },
      ],
      'import-x/no-anonymous-default-export': [
        'error',
        {
          allowArray: true,
          allowArrowFunction: true,
          allowAnonymousClass: true,
          allowAnonymousFunction: true,
          allowCallExpression: true,
          allowLiteral: true,
          allowObject: true,
        },
      ],
      'react/react-in-jsx-scope': 'off',
      'react/prop-types': 'off',
      'react/jsx-no-target-blank': ['error', { allowReferrer: true }],
    },
  },
  {
    files: ['server/**/*.ts'],
    rules: {
      'import-x/no-unresolved': 'off',
    },
  },
];
