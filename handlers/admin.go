package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminMiddleware checks if the user is an admin
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
			c.Abort()
			return
		}

		// Check if user is admin
		var role string
		query := `SELECT role FROM users WHERE id = $1`
		err := DB.QueryRow(query, userID).Scan(&role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check user role"})
			c.Abort()
			return
		}

		if role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdminOrEmployeeMiddleware checks if the user is an admin or employee
func AdminOrEmployeeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
			c.Abort()
			return
		}

		// Check if user is admin or employee
		var role string
		query := `SELECT role FROM users WHERE id = $1`
		err := DB.QueryRow(query, userID).Scan(&role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check user role"})
			c.Abort()
			return
		}

		if role != "admin" && role != "employee" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin or Employee access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdminDashboard serves the admin dashboard HTML
func AdminDashboard(c *gin.Context) {
	html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>FMBQ Admin Dashboard</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://unpkg.com/alpinejs@3.x.x/dist/cdn.min.js" defer></script>
</head>
<body class="bg-gray-100">
    <div class="min-h-screen" x-data="adminDashboard()">
        <!-- Sidebar -->
        <div class="fixed inset-y-0 left-0 z-50 w-64 bg-gray-800 text-white transform -translate-x-full transition-transform duration-300 ease-in-out" 
             :class="sidebarOpen ? 'translate-x-0' : ''" x-ref="sidebar">
            <div class="flex items-center justify-center h-16 bg-gray-900">
                <h1 class="text-xl font-bold">FMBQ Admin</h1>
            </div>
            <nav class="mt-8">
                <a href="#" @click="activeTab = 'dashboard'" 
                   :class="activeTab === 'dashboard' ? 'bg-gray-700' : ''"
                   class="flex items-center px-6 py-3 text-gray-300 hover:bg-gray-700">
                    <span class="ml-3">Dashboard</span>
                </a>
                <a href="#" @click="activeTab = 'products'" 
                   :class="activeTab === 'products' ? 'bg-gray-700' : ''"
                   class="flex items-center px-6 py-3 text-gray-300 hover:bg-gray-700">
                    <span class="ml-3">Products</span>
                </a>
                <a href="#" @click="activeTab = 'categories'" 
                   :class="activeTab === 'categories' ? 'bg-gray-700' : ''"
                   class="flex items-center px-6 py-3 text-gray-300 hover:bg-gray-700">
                    <span class="ml-3">Categories</span>
                </a>
                <a href="#" @click="activeTab = 'brands'" 
                   :class="activeTab === 'brands' ? 'bg-gray-700' : ''"
                   class="flex items-center px-6 py-3 text-gray-300 hover:bg-gray-700">
                    <span class="ml-3">Brands</span>
                </a>
                <a href="#" @click="activeTab = 'users'" 
                   :class="activeTab === 'users' ? 'bg-gray-700' : ''"
                   class="flex items-center px-6 py-3 text-gray-300 hover:bg-gray-700">
                    <span class="ml-3">Users</span>
                </a>
                <a href="#" @click="activeTab = 'orders'" 
                   :class="activeTab === 'orders' ? 'bg-gray-700' : ''"
                   class="flex items-center px-6 py-3 text-gray-300 hover:bg-gray-700">
                    <span class="ml-3">Orders</span>
                </a>
            </nav>
        </div>

        <!-- Main Content -->
        <div class="ml-0 transition-all duration-300" :class="sidebarOpen ? 'ml-64' : ''">
            <!-- Header -->
            <header class="bg-white shadow-sm border-b">
                <div class="flex items-center justify-between px-6 py-4">
                    <button @click="sidebarOpen = !sidebarOpen" class="text-gray-600 hover:text-gray-900">
                        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"></path>
                        </svg>
                    </button>
                    <div class="flex items-center space-x-4">
                        <span class="text-sm text-gray-600">Welcome, Admin</span>
                        <button @click="logout()" class="bg-red-600 text-white px-4 py-2 rounded hover:bg-red-700">
                            Logout
                        </button>
                    </div>
                </div>
            </header>

            <!-- Content Area -->
            <main class="p-6">
                <!-- Dashboard Tab -->
                <div x-show="activeTab === 'dashboard'" class="space-y-6">
                    <h2 class="text-2xl font-bold text-gray-900">Dashboard Overview</h2>
                    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                        <div class="bg-white p-6 rounded-lg shadow">
                            <h3 class="text-lg font-semibold text-gray-900">Total Products</h3>
                            <p class="text-3xl font-bold text-blue-600" x-text="stats.totalProducts">0</p>
                        </div>
                        <div class="bg-white p-6 rounded-lg shadow">
                            <h3 class="text-lg font-semibold text-gray-900">Total Users</h3>
                            <p class="text-3xl font-bold text-green-600" x-text="stats.totalUsers">0</p>
                        </div>
                        <div class="bg-white p-6 rounded-lg shadow">
                            <h3 class="text-lg font-semibold text-gray-900">Total Orders</h3>
                            <p class="text-3xl font-bold text-purple-600" x-text="stats.totalOrders">0</p>
                        </div>
                        <div class="bg-white p-6 rounded-lg shadow">
                            <h3 class="text-lg font-semibold text-gray-900">Revenue</h3>
                            <p class="text-3xl font-bold text-orange-600" x-text="stats.revenue">$0</p>
                        </div>
                    </div>
                </div>

                <!-- Products Tab -->
                <div x-show="activeTab === 'products'" class="space-y-6">
                    <div class="flex justify-between items-center">
                        <h2 class="text-2xl font-bold text-gray-900">Products Management</h2>
                        <button @click="showProductModal = true" class="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700">
                            Add Product
                        </button>
                    </div>
                    <div class="bg-white rounded-lg shadow overflow-hidden">
                        <table class="min-w-full divide-y divide-gray-200">
                            <thead class="bg-gray-50">
                                <tr>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Brand</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                                </tr>
                            </thead>
                            <tbody class="bg-white divide-y divide-gray-200">
                                <template x-for="product in products" :key="product.id">
                                    <tr>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900" x-text="product.title"></td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500" x-text="product.brand_name"></td>
                                        <td class="px-6 py-4 whitespace-nowrap">
                                            <span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full" 
                                                  :class="product.is_active ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'"
                                                  x-text="product.is_active ? 'Active' : 'Inactive'"></span>
                                        </td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm font-medium">
                                            <button @click="editProduct(product)" class="text-indigo-600 hover:text-indigo-900 mr-3">Edit</button>
                                            <button @click="deleteProduct(product.id)" class="text-red-600 hover:text-red-900">Delete</button>
                                        </td>
                                    </tr>
                                </template>
                            </tbody>
                        </table>
                    </div>
                </div>

                <!-- Categories Tab -->
                <div x-show="activeTab === 'categories'" class="space-y-6">
                    <div class="flex justify-between items-center">
                        <h2 class="text-2xl font-bold text-gray-900">Categories Management</h2>
                        <button @click="showCategoryModal = true" class="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700">
                            Add Category
                        </button>
                    </div>
                    <div class="bg-white rounded-lg shadow overflow-hidden">
                        <table class="min-w-full divide-y divide-gray-200">
                            <thead class="bg-gray-50">
                                <tr>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Slug</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Parent</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                                </tr>
                            </thead>
                            <tbody class="bg-white divide-y divide-gray-200">
                                <template x-for="category in categories" :key="category.id">
                                    <tr>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900" x-text="category.name"></td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500" x-text="category.slug"></td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500" x-text="category.parent_name || 'Root'"></td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm font-medium">
                                            <button @click="editCategory(category)" class="text-indigo-600 hover:text-indigo-900 mr-3">Edit</button>
                                            <button @click="deleteCategory(category.id)" class="text-red-600 hover:text-red-900">Delete</button>
                                        </td>
                                    </tr>
                                </template>
                            </tbody>
                        </table>
                    </div>
                </div>

                <!-- Brands Tab -->
                <div x-show="activeTab === 'brands'" class="space-y-6">
                    <div class="flex justify-between items-center">
                        <h2 class="text-2xl font-bold text-gray-900">Brands Management</h2>
                        <button @click="showBrandModal = true" class="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700">
                            Add Brand
                        </button>
                    </div>
                    <div class="bg-white rounded-lg shadow overflow-hidden">
                        <table class="min-w-full divide-y divide-gray-200">
                            <thead class="bg-gray-50">
                                <tr>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Slug</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Description</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                                </tr>
                            </thead>
                            <tbody class="bg-white divide-y divide-gray-200">
                                <template x-for="brand in brands" :key="brand.id">
                                    <tr>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900" x-text="brand.name"></td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500" x-text="brand.slug || '-'"></td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500" x-text="brand.description || '-'"></td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm font-medium">
                                            <button @click="editBrand(brand)" class="text-indigo-600 hover:text-indigo-900 mr-3">Edit</button>
                                            <button @click="deleteBrand(brand.id)" class="text-red-600 hover:text-red-900">Delete</button>
                                        </td>
                                    </tr>
                                </template>
                            </tbody>
                        </table>
                    </div>
                </div>

                <!-- Users Tab -->
                <div x-show="activeTab === 'users'" class="space-y-6">
                    <h2 class="text-2xl font-bold text-gray-900">Users Management</h2>
                    <div class="bg-white rounded-lg shadow overflow-hidden">
                        <table class="min-w-full divide-y divide-gray-200">
                            <thead class="bg-gray-50">
                                <tr>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Email</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Phone</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Role</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                                </tr>
                            </thead>
                            <tbody class="bg-white divide-y divide-gray-200">
                                <template x-for="user in users" :key="user.id">
                                    <tr>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900" x-text="user.full_name || 'N/A'"></td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500" x-text="user.email || 'N/A'"></td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500" x-text="user.phone || 'N/A'"></td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500" x-text="user.role"></td>
                                        <td class="px-6 py-4 whitespace-nowrap">
                                            <span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full" 
                                                  :class="user.is_active ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'"
                                                  x-text="user.is_active ? 'Active' : 'Inactive'"></span>
                                        </td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm font-medium">
                                            <button @click="editUser(user)" class="text-indigo-600 hover:text-indigo-900 mr-3">Edit</button>
                                            <button @click="toggleUserStatus(user.id, !user.is_active)" class="text-yellow-600 hover:text-yellow-900">
                                                <span x-text="user.is_active ? 'Deactivate' : 'Activate'"></span>
                                            </button>
                                        </td>
                                    </tr>
                                </template>
                            </tbody>
                        </table>
                    </div>
                </div>

                <!-- Orders Tab -->
                <div x-show="activeTab === 'orders'" class="space-y-6">
                    <h2 class="text-2xl font-bold text-gray-900">Orders Management</h2>
                    <div class="bg-white rounded-lg shadow overflow-hidden">
                        <table class="min-w-full divide-y divide-gray-200">
                            <thead class="bg-gray-50">
                                <tr>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Order #</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Customer</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Total</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Date</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                                </tr>
                            </thead>
                            <tbody class="bg-white divide-y divide-gray-200">
                                <template x-for="order in orders" :key="order.id">
                                    <tr>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900" x-text="order.order_number"></td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500" x-text="order.customer_name || 'N/A'"></td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500" x-text="order.total_amount + ' ' + order.currency"></td>
                                        <td class="px-6 py-4 whitespace-nowrap">
                                            <span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full" 
                                                  :class="getStatusColor(order.status)"
                                                  x-text="order.status"></span>
                                        </td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500" x-text="formatDate(order.created_at)"></td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm font-medium">
                                            <button @click="viewOrder(order)" class="text-indigo-600 hover:text-indigo-900 mr-3">View</button>
                                            <button @click="updateOrderStatus(order.id, order.status)" class="text-green-600 hover:text-green-900">Update Status</button>
                                        </td>
                                    </tr>
                                </template>
                            </tbody>
                        </table>
                    </div>
                </div>
            </main>
        </div>
    </div>

    <script>
        function adminDashboard() {
            return {
                sidebarOpen: false,
                activeTab: 'dashboard',
                showProductModal: false,
                showCategoryModal: false,
                showBrandModal: false,
                stats: {
                    totalProducts: 0,
                    totalUsers: 0,
                    totalOrders: 0,
                    revenue: '$0'
                },
                products: [],
                categories: [],
                brands: [],
                users: [],
                orders: [],
                
                init() {
                    this.loadStats();
                    this.loadProducts();
                    this.loadCategories();
                    this.loadBrands();
                    this.loadUsers();
                    this.loadOrders();
                },
                
                async loadStats() {
                    try {
                        const response = await fetch('/api/v1/admin/stats');
                        const data = await response.json();
                        this.stats = data;
                    } catch (error) {
                        console.error('Error loading stats:', error);
                    }
                },
                
                async loadProducts() {
                    try {
                        const response = await fetch('/api/v1/products');
                        const data = await response.json();
                        this.products = data.products || [];
                    } catch (error) {
                        console.error('Error loading products:', error);
                    }
                },
                
                async loadCategories() {
                    try {
                        const response = await fetch('/api/v1/categories');
                        const data = await response.json();
                        this.categories = data.categories || [];
                    } catch (error) {
                        console.error('Error loading categories:', error);
                    }
                },
                
                async loadBrands() {
                    try {
                        const response = await fetch('/api/v1/brands');
                        const data = await response.json();
                        this.brands = data.brands || [];
                    } catch (error) {
                        console.error('Error loading brands:', error);
                    }
                },
                
                async loadUsers() {
                    try {
                        const response = await fetch('/api/v1/admin/users');
                        const data = await response.json();
                        this.users = data.users || [];
                    } catch (error) {
                        console.error('Error loading users:', error);
                    }
                },
                
                async loadOrders() {
                    try {
                        const response = await fetch('/api/v1/admin/orders');
                        const data = await response.json();
                        this.orders = data.orders || [];
                    } catch (error) {
                        console.error('Error loading orders:', error);
                    }
                },
                
                getStatusColor(status) {
                    const colors = {
                        'created': 'bg-blue-100 text-blue-800',
                        'paid': 'bg-green-100 text-green-800',
                        'shipped': 'bg-purple-100 text-purple-800',
                        'delivered': 'bg-gray-100 text-gray-800',
                        'cancelled': 'bg-red-100 text-red-800'
                    };
                    return colors[status] || 'bg-gray-100 text-gray-800';
                },
                
                formatDate(dateString) {
                    return new Date(dateString).toLocaleDateString();
                },
                
                logout() {
                    localStorage.removeItem('token');
                    window.location.href = '/';
                }
            }
        }
    </script>
</body>
</html>
	`
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}
