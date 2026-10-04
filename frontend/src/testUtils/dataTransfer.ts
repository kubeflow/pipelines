/**
 * Builds the `dataTransfer` payload for a synthetic drop event.
 *
 * react-dropzone reads dropped files through file-selector, which for a `drop`
 * event takes them from `DataTransfer.items` and never looks at
 * `DataTransfer.files`. Real browsers populate both, so a stub carrying only
 * `files` silently yields no files.
 *
 * Each item is the minimal shape file-selector needs: `kind: 'file'` so it
 * survives filtering, and `getAsFile()` to read the File. Omitting
 * `webkitGetAsEntry` and `getAsFileSystemHandle` keeps it on the plain-file
 * path rather than directory traversal or the File System Access API.
 */
export function dataTransferWithFiles(...files: File[]) {
  return {
    files,
    items: files.map((file) => ({ kind: 'file', type: file.type, getAsFile: () => file })),
    types: ['Files'],
  };
}
