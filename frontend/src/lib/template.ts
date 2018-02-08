export default interface Template {
  author: string;
  description: string;
  id: number;
  location: string;
  name: string;
  parameters: { name: string, description: string }[];
  sharedWith: string;
  tags: string[];
}
