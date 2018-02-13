interface Config {
  api: string;
  ftp: string;
}

export const configs = new Map<string, Config>([
  [
    'dev', {
      api: '/_api',
      ftp: '/_ftp',
    },
  ],
  [
    'prod', {
      api: '',
      ftp: '',
    }
  ]
]);

export const config = configs.get(process.env.NODE_ENV || 'dev') as Config;
