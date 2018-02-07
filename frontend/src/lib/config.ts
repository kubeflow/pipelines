const config = {
  production: {
    api: '',
  },
  dev: {
    api: 'http://localhost:3001',
  },
}

export default (config as any)[process.env.NODE_ENV || 'dev'];
