export enum ArtifactType {
  LOCAL,
  GCS,
}

export interface Artifact {
  location: string;
  type: ArtifactType;
  viewer?: any;
}
