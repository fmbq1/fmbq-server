# Testing the Role System

## Prerequisites

1. Make sure the Go server is running on port 8080
2. Make sure the Next.js admin dashboard is running
3. Have at least one admin user created

## Steps to Test

### 1. Create an Admin User (if not exists)

```bash
# Using the admin signup endpoint
curl -X POST https://fmbq-server.onrender.com/admin/signup \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+22212345678",
    "password": "admin123",
    "full_name": "Admin User",
    "email": "admin@example.com"
  }'
```

### 2. Login as Admin

1. Go to `http://192.168.100.9:3000/admin/login`
2. Enter the admin credentials
3. You should be redirected to the admin dashboard

### 3. Test Users Management

1. Navigate to `http://192.168.100.9:3000/admin/users`
2. You should see the users management page
3. Check browser console for any errors
4. The page should show:
   - User statistics cards
   - Users table with role management
   - Role selection dropdown (admin, employee, user)
   - Status toggle buttons

### 4. Test Role Changes

1. Try changing a user's role using the dropdown
2. Try toggling user status (activate/deactivate)
3. Check if changes are reflected in the UI

### 5. Test API Endpoints Directly

```bash
# Get all users (requires admin token)
curl -X GET https://fmbq-server.onrender.com/api/v1/admin/users \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"

# Get user statistics
curl -X GET https://fmbq-server.onrender.com/api/v1/admin/users-stats \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"

# Update user role
curl -X PUT https://fmbq-server.onrender.com/api/v1/admin/users/USER_ID/role \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"role": "employee"}'
```

## Troubleshooting

### If "No users found" appears:

1. Check if you're logged in as admin (check localStorage for `admin_token`)
2. Check browser console for API errors
3. Verify the Go server is running and accessible
4. Check if there are any users in the database

### If authentication fails:

1. Make sure you're using the admin login page (`/admin/login`)
2. Check that the admin user exists in the database
3. Verify the admin token is stored in localStorage

### If API calls fail:

1. Check the Go server logs for errors
2. Verify the database connection
3. Check if the admin middleware is working correctly
4. Ensure the user has admin role in the database

## Expected Behavior

- **Admin users** can access `/admin/users` and see all users
- **Role changes** should be reflected immediately in the UI
- **Status toggles** should work for activating/deactivating users
- **Statistics** should show correct counts by role
- **Only admins** can access user management features
