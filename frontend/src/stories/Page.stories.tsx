import { Meta, StoryObj } from '@storybook/react';
import { fn } from 'storybook/test';

import { Page } from './Page';

const meta: Meta<typeof Page> = {
  title: 'Example/Page',
  component: Page,
  args: {
    onLogin: fn(),
    onLogout: fn(),
    onCreateAccount: fn(),
  },
};

export default meta;
type Story = StoryObj<typeof Page>;

export const LoggedIn: Story = {
  args: {
    user: {},
  },
};

export const LoggedOut: Story = {
  args: {},
};
