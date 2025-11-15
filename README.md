# FMBQ Server

A modern e-commerce platform server built with Go, designed for the Mauritanian market but open to international brands.

## Features

- **Phone Number Authentication**: Users authenticate using only their phone number with OTP verification
- **Product Management**: Complete product catalog with models, colors, sizes, and SKUs
- **Inventory Management**: Real-time inventory tracking with reservations
- **Shopping Cart**: Full cart functionality with persistent storage
- **Order Management**: Complete order lifecycle management
- **Loyalty System**: Points-based loyalty program
- **Image Management**: Cloudinary integration for product images
- **PostgreSQL Database**: Robust data storage with proper relationships

## Tech Stack

- **Backend**: Go with Gin framework
- **Database**: PostgreSQL with pgcrypto extension
- **Image Storage**: Cloudinary
- **Authentication**: JWT tokens
- **Environment**: Configurable via environment variables

## Database Schema

The application uses a comprehensive database schema with the following main entities:

- **Categories**: Hierarchical product categories
- **Brands**: Product brands and manufacturers
- **Product Models**: Abstract products (e.g., "Nike Air 2025")
- **Product Colors**: Color variants of products
- **SKUs**: Specific sellable items (color + size combinations)
- **Inventory**: Stock management
- **Prices**: Pricing with support for sales and promotions
- **Users**: Customer accounts (phone-based authentication)
- **Orders**: Order management with items
- **Carts**: Shopping cart functionality
- **Reviews**: Product reviews and ratings
- **Loyalty**: Points-based loyalty system

## Setup Instructions

### Prerequisites

1. Go 1.21 or higher
2. PostgreSQL 12 or higher
3. Cloudinary account (optional, for image management)

### Installation

1. **Clone the repository**

   ```bash
   git clone <repository-url>
   cd fmbq-server
   ```

2. **Install dependencies**

   ```bash
   go mod tidy
   ```

3. **Set up environment variables**

   ```bash
   cp env.example .env
   # Edit .env with your configuration
   ```

4. **Configure your database**

   - Create a PostgreSQL database
   - Update `DATABASE_URL` in your `.env` file
   - Example: `postgres://username:password@localhost:5432/fmbq_db?sslmode=disable`

5. **Configure Cloudinary (optional)**

   - Get your Cloudinary URL from your dashboard
   - Update `CLOUDINARY_URL` in your `.env` file
   - Example: `cloudinary://api_key:api_secret@cloud_name`

6. **Run the server**
   ```bash
   go run main.go
   ```

The server will automatically:

- Connect to the database
- Create all necessary tables
- Initialize Cloudinary (if configured)
- Start the API server

### Environment Variables

| Variable         | Description                          | Default                                                      |
| ---------------- | ------------------------------------ | ------------------------------------------------------------ |
| `DATABASE_URL`   | PostgreSQL connection string         | `postgres://user:password@localhost/fmbq_db?sslmode=disable` |
| `CLOUDINARY_URL` | Cloudinary configuration URL         | (empty)                                                      |
| `JWT_SECRET`     | JWT signing secret                   | `your-secret-key-change-in-production`                       |
| `PORT`           | Server port                          | `8080`                                                       |
| `ENVIRONMENT`    | Environment (development/production) | `development`                                                |

## API Endpoints

### Authentication

- `POST /api/v1/auth/send-otp` - Send OTP to phone number
- `POST /api/v1/auth/verify-otp` - Verify OTP and get JWT token
- `POST /api/v1/auth/refresh` - Refresh JWT token

### Products

- `GET /api/v1/products` - List products (with filtering)
- `GET /api/v1/products/:id` - Get product details
- `POST /api/v1/products` - Create product (admin)
- `PUT /api/v1/products/:id` - Update product (admin)
- `DELETE /api/v1/products/:id` - Delete product (admin)

### Categories

- `GET /api/v1/categories` - List categories
- `GET /api/v1/categories/:id` - Get category details
- `POST /api/v1/categories` - Create category (admin)
- `PUT /api/v1/categories/:id` - Update category (admin)
- `DELETE /api/v1/categories/:id` - Delete category (admin)

### Brands

- `GET /api/v1/brands` - List brands
- `GET /api/v1/brands/:id` - Get brand details
- `POST /api/v1/brands` - Create brand (admin)
- `PUT /api/v1/brands/:id` - Update brand (admin)
- `DELETE /api/v1/brands/:id` - Delete brand (admin)

### User (Protected)

- `GET /api/v1/users/profile` - Get user profile
- `PUT /api/v1/users/profile` - Update user profile
- `GET /api/v1/users/orders` - Get user orders

### Cart (Protected)

- `GET /api/v1/cart` - Get cart contents
- `POST /api/v1/cart/add` - Add item to cart
- `PUT /api/v1/cart/update` - Update cart item
- `DELETE /api/v1/cart/remove/:id` - Remove item from cart
- `DELETE /api/v1/cart/clear` - Clear cart

### Orders (Protected)

- `POST /api/v1/orders` - Create order
- `GET /api/v1/orders/:id` - Get order details
- `PUT /api/v1/orders/:id/cancel` - Cancel order

## Authentication

The API uses phone number-based authentication:

1. User sends phone number to `/api/v1/auth/send-otp`
2. Server generates and stores OTP (in development, OTP is returned in response)
3. User sends phone number and OTP to `/api/v1/auth/verify-otp`
4. Server returns JWT token and user information
5. User includes JWT token in Authorization header for protected endpoints

## Database Initialization

The server automatically creates all necessary tables on startup. The initialization order respects foreign key dependencies:

1. Categories
2. Brands
3. Product Models
4. Product Model Categories (junction table)
5. Size Charts
6. Product Colors
7. SKUs
8. Product Images
9. Inventory
10. Prices
11. Users
12. Loyalty Accounts
13. Loyalty Transactions
14. Addresses
15. Orders
16. Order Items
17. Carts
18. Cart Items
19. Reviews

## Development

### Running in Development Mode

```bash
go run main.go
```

The server will run on `https://fmbq-server.onrender.com` by default.

### Health Check

Visit `https://fmbq-server.onrender.com/health` to check if the server is running.

## Production Deployment

1. Set `ENVIRONMENT=production` in your environment variables
2. Use a strong `JWT_SECRET`
3. Configure proper CORS origins
4. Use a production PostgreSQL database
5. Set up proper logging and monitoring

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License.
"# fmbq-server"
"# fmbq-server"
"# fmbq-admn"
"# fmbq-admn"
