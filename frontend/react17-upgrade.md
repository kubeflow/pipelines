# React 17 Upgrade Guide

## 📋 Pre-Upgrade Checklist

1. **Backup your current codebase**
   ```bash
   git checkout -b react17-upgrade
   git add -A && git commit -m "Pre-React 17 upgrade backup"
   ```

2. **Run current tests to establish baseline**
   ```bash
   npm test -- --coverage --watchAll=false
   ```

## 🚀 Upgrade Steps

### Step 1: Install Updated Dependencies

```bash
# Navigate to frontend directory
cd frontend

# Update React core
npm install react@^17.0.2 react-dom@^17.0.2

# Update TypeScript types
npm install --save-dev @types/react@^17.0.0 @types/react-dom@^17.0.0 @types/react-test-renderer@^17.0.0

# Update testing dependencies
npm uninstall enzyme-adapter-react-16
npm install --save-dev enzyme-adapter-react-17@^1.0.6 react-test-renderer@^17.0.2
```

### Step 2: Update Critical Dependencies

**⚠️ IMPORTANT: Material-UI v3 is not compatible with React 17**

```bash
# Remove old Material-UI (this will require code changes)
npm uninstall @material-ui/core @material-ui/icons

# Install Material-UI v4 (React 17 compatible)
npm install @mui/material@^5.0.0 @mui/icons-material@^5.0.0
# OR stick with Material-UI v4 for easier migration:
npm install @material-ui/core@^4.12.4 @material-ui/icons@^4.11.3
```

**Optional: Update React Router (recommended)**
```bash
npm install react-router-dom@^6.2.0
npm install --save-dev @types/react-router-dom@^5.3.0
```

### Step 3: Verify Configuration Files

✅ Already updated:
- `tsconfig.json` - JSX transform updated to `"react-jsx"`
- `setupTests.ts` - Enzyme adapter updated to React 17

### Step 4: Test the Upgrade

```bash
# Clear node modules and reinstall
rm -rf node_modules package-lock.json
npm install

# Run tests
npm test -- --watchAll=false

# Run build
npm run build

# Start development server
npm start
```

### Step 5: Address Material-UI Breaking Changes

If upgrading to Material-UI v4, you'll need to update imports:

**Before (v3):**
```tsx
import { Button, TextField } from '@material-ui/core';
import AddIcon from '@material-ui/icons/Add';
```

**After (v4):**
```tsx
import { Button, TextField } from '@material-ui/core';
import AddIcon from '@material-ui/icons/Add';
// Most imports stay the same, but some component APIs changed
```

### Step 6: Optional React Router v6 Migration

If upgrading React Router, major changes include:
- `Switch` → `Routes`
- `Route` component changes
- `useHistory` → `useNavigate`

## 🧪 Testing Strategy

### Test in this order:
1. **Unit tests**: `npm test`
2. **Integration tests**: `npm run test:ci`
3. **Build process**: `npm run build`
4. **Development server**: `npm start`
5. **Storybook**: `npm run storybook`

### Common Issues & Solutions

#### Issue: Enzyme tests failing
**Solution**: Make sure enzyme-adapter-react-17 is properly installed and configured

#### Issue: Material-UI styling broken
**Solution**: Material-UI v4 has theme changes. Update theme provider

#### Issue: TypeScript errors
**Solution**: Update @types packages and check for deprecated APIs

## 🎯 Benefits of React 17

1. **Gradual Upgrades**: Easier to upgrade to React 18 later
2. **New JSX Transform**: No need to import React in every JSX file
3. **Better Event Delegation**: Events now attach to root container
4. **Improved Error Boundaries**: Better error handling
5. **Performance**: Various under-the-hood improvements

## 🚨 Rollback Plan

If issues arise:
```bash
git checkout react-v7-upgrade  # your original branch
npm install
```

## 📝 Notes

- The current setup already uses React Scripts 5.0.0 which supports React 17
- Most of your code patterns are already React 17 compatible
- Focus on Material-UI upgrade as the main potential issue
- Window/document event listeners found in codebase should work unchanged 