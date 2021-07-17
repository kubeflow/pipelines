export interface Feature {
  name: string;
  description: string;
  active: boolean;
}

export const features: Feature[] = [
  {
    name: 'v2',
    description: 'Show v2 features',
    active: false,
  },
];
