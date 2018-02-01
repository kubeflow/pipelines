export default class Utils {
  public static deleteAllChildren(parent: HTMLElement) {
    while (parent.firstChild) {
      parent.removeChild(parent.firstChild);
    }
  }
}
