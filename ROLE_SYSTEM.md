# Role-Based Access Control System

This document describes the role-based access control system implemented in the FMBQ application.

## Roles

The system supports three main roles:

### 1. Admin (`admin`)

- **Full system access**
- Can manage all users, products, orders, categories, and brands
- Can change user roles and status
- Can access all admin endpoints
- Can view and modify all data

### 2. Employee (`employee`)

- **Limited admin access**
- Can view and manage products, orders, categories, and brands
- Cannot manage user roles or access user management features
- Cannot access sensitive admin functions
- Can process orders and manage inventory

### 3. User (`user`)

- **Customer access**
- Can view products and place orders
- Can manage their own profile and cart
- Cannot access admin features
- Default role for new registrations

## API Endpoints

### Admin-Only Endpoints

- `GET /api/v1/admin/users` - List all users
- `GET /api/v1/admin/users/:id` - Get user details
- `PUT /api/v1/admin/users/:id/role` - Update user role
- `PUT /api/v1/admin/users/:id/status` - Toggle user status
- `PUT /api/v1/admin/users/:id/profile` - Update user profile
- `GET /api/v1/admin/users-stats` - Get user statistics

### Admin or Employee Endpoints

- `GET /api/v1/admin/products` - List products (admin view)
- `GET /api/v1/admin/orders` - List orders
- `PUT /api/v1/admin/orders/:id/status` - Update order status
- All product, category, and brand management endpoints

### User Endpoints

- `GET /api/v1/products` - View products
- `GET /api/v1/users/profile` - Get own profile
- `PUT /api/v1/users/profile` - Update own profile
- Cart and order management endpoints

## Middleware

### AuthMiddleware

- Validates JWT token
- Sets `user_id` and `user_role` in context
- Required for all protected endpoints

### AdminMiddleware

- Checks if user role is `admin`
- Required for admin-only endpoints

### AdminOrEmployeeMiddleware

- Checks if user role is `admin` or `employee`
- Required for admin/employee endpoints

## Database Schema

The `users` table includes:

- `role` field with values: `admin`, `employee`, `user`
- `is_active` field for account status
- Default role for new users: `user`

## Frontend Implementation

### Admin Dashboard

- Users management page at `/admin/users`
- Role selection dropdown with three options
- Status toggle for user activation/deactivation
- Statistics dashboard showing user counts by role

### Role Management

- Admins can change user roles through the UI
- Role changes are immediately reflected in the system
- Proper validation ensures only valid roles are accepted

## Security Considerations

1. **Token Validation**: All requests include JWT token validation
2. **Role Verification**: Server-side role checking for all admin endpoints
3. **Input Validation**: Role values are validated against allowed values
4. **Audit Trail**: All role changes are logged and can be tracked

## Usage Examples

### Creating an Admin User

```sql
INSERT INTO users (id, email, full_name, role, is_active)
VALUES (gen_random_uuid(), 'admin@example.com', 'Admin User', 'admin', true);
```

### Updating User Role via API

```bash
curl -X PUT http://192.168.0.131:8080/api/v1/admin/users/{user_id}/role \
  -H "Authorization: Bearer {admin_token}" \
  -H "Content-Type: application/json" \
  -d '{"role": "employee"}'
```

### Checking User Role in Middleware

```go
role, exists := c.Get("user_role")
if !exists || role != "admin" {
    c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
    c.Abort()
    return
}
```

## Future Enhancements

1. **Permission System**: More granular permissions within roles
2. **Role Hierarchy**: Define role inheritance and permissions
3. **Audit Logging**: Track all role changes and admin actions
4. **Role Expiration**: Temporary role assignments with expiration
5. **Multi-tenant Support**: Role management per organization
