export default () => {
  // This let unit tests run in UTC timezone consistently, despite developers'
  // dev machine's timezone.
  // Reference: https://stackoverflow.com/a/56482581
  process.env.TZ = 'UTC';
};
