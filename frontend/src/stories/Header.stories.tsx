import { Meta, StoryObj } from '@storybook/react';
import { fn } from 'storybook/test';

import { Header } from './Header';

const meta: Meta<typeof Header> = {
  title: 'Example/Header',
  component: Header,
  args: {
    onLogin: fn(),
    onLogout: fn(),
    onCreateAccount: fn(),
  },
};

export default meta;
type Story = StoryObj<typeof Header>;

export const LoggedIn: Story = {
  args: {
    user: {},
  },
};

export const LoggedOut: Story = {
  args: {},
};
