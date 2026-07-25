const eslint = require('@eslint/js');
const typescriptEslint = require('@typescript-eslint/eslint-plugin');
const typescriptParser = require('@typescript-eslint/parser');
const globals = require('globals');
const { createNodeResolver, importX } = require('eslint-plugin-import-x');
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
      'react-hooks': reactHooks,
    },
    settings: {
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
      // React Hooks 7 adds React Compiler diagnostics to its recommended preset.
      // Enable correctness rules independently of compiler adoption.
      'react-hooks/rules-of-hooks': 'error',
      'react-hooks/exhaustive-deps': 'warn',
      'react-hooks/static-components': 'error',
      'react-hooks/use-memo': 'error',
      'react-hooks/incompatible-library': 'warn',
      'react-hooks/immutability': 'error',
      'react-hooks/globals': 'error',
      'react-hooks/refs': 'error',
      'react-hooks/set-state-in-effect': 'warn',
      'react-hooks/error-boundaries': 'error',
      'react-hooks/purity': 'error',
      'react-hooks/set-state-in-render': 'error',
      'react-hooks/unsupported-syntax': 'warn',
      'react-hooks/preserve-manual-memoization': 'warn',
      // These rules only become actionable when React Compiler is configured.
      'react-hooks/config': 'off',
      'react-hooks/gating': 'off',
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
    },
  },
  {
    files: ['server/**/*.ts'],
    rules: {
      'import-x/no-unresolved': 'off',
    },
  },
];
