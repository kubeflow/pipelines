export default class Utils {
  static log() {
    console.log('hi');
  }
  public static deleteAllChildren(parent: HTMLElement) {
    while (parent.firstChild) {
      parent.removeChild(parent.firstChild);
    }
  }
}
