const config = {
  production: {
    api: '',
    ftp: '',
  },
  dev: {
    api: '/_api',
    ftp: '/_ftp',
  },
}

export default (config as any)[process.env.NODE_ENV || 'dev'];
