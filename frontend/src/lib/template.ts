export interface Template {
  author: string;
  description: string;
  id: number;
  location: string;
  name: string;
  parameters: Array<{ description: string, name: string }>;
  sharedWith: string;
  tags: string[];
}
