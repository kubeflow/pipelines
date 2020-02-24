module.exports = {
  test(val) {
    return val && !!val._isMockFunction;
  },
  print(val) {
    return '[MockFunction]';
  },
};
