# Customer Relationship Management (CRM) System

This document describes the comprehensive CRM system implemented in the FMBQ application, designed to manage customer relationships, interactions, and loyalty programs similar to Odoo.

## 🎯 **System Overview**

The CRM system provides a complete customer management solution with:

- **Customer Management**: Individual and business customer profiles
- **Interaction Tracking**: Complete history of customer communications
- **Loyalty Program**: Points-based rewards system with tiers
- **Analytics**: Customer insights and performance metrics
- **Segmentation**: Customer grouping and targeting

## 🏗️ **Database Schema**

### Core Tables

#### 1. Customers Table

```sql
CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    company_name TEXT,
    contact_name TEXT,
    email TEXT,
    phone TEXT,
    address TEXT,
    city TEXT,
    state TEXT,
    country TEXT,
    postal_code TEXT,
    customer_type TEXT DEFAULT 'individual' CHECK (customer_type IN ('individual', 'business')),
    status TEXT DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'prospect')),
    source TEXT DEFAULT 'website',
    tags JSONB DEFAULT '[]',
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    last_contact TIMESTAMP WITH TIME ZONE
);
```

#### 2. Customer Interactions Table

```sql
CREATE TABLE customer_interactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('call', 'email', 'meeting', 'support', 'sale', 'follow_up', 'other')),
    subject TEXT NOT NULL,
    description TEXT,
    outcome TEXT,
    priority TEXT DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'cancelled')),
    duration INTEGER, -- in minutes
    follow_up TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);
```

#### 3. Customer Segments Table

```sql
CREATE TABLE customer_segments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    criteria JSONB NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);
```

#### 4. Enhanced Loyalty System

```sql
CREATE TABLE loyalty_accounts (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    points_balance BIGINT DEFAULT 0,
    tier TEXT DEFAULT 'bronze' CHECK (tier IN ('bronze', 'silver', 'gold', 'platinum')),
    total_earned BIGINT DEFAULT 0,
    total_redeemed BIGINT DEFAULT 0,
    last_activity TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);
```

## 🔌 **API Endpoints**

### Customer Management

- `GET /api/v1/admin/crm/customers` - List all customers with filtering
- `GET /api/v1/admin/crm/customers/:id` - Get specific customer
- `POST /api/v1/admin/crm/customers` - Create new customer
- `PUT /api/v1/admin/crm/customers/:id` - Update customer
- `DELETE /api/v1/admin/crm/customers/:id` - Delete customer

### Customer Interactions

- `GET /api/v1/admin/crm/customers/:customer_id/interactions` - Get customer interactions
- `POST /api/v1/admin/crm/customers/:customer_id/interactions` - Create interaction
- `PUT /api/v1/admin/crm/interactions/:id` - Update interaction
- `DELETE /api/v1/admin/crm/interactions/:id` - Delete interaction

### Analytics & Statistics

- `GET /api/v1/admin/crm/stats` - Overall CRM statistics
- `GET /api/v1/admin/crm/customers/:customer_id/stats` - Customer-specific stats

## 🎨 **Frontend Features**

### CRM Dashboard (`/admin/crm`)

- **Customer Overview**: Complete customer list with search and filtering
- **Interaction Tracking**: Timeline of all customer communications
- **Analytics**: Customer insights and performance metrics
- **Quick Actions**: Fast access to common customer tasks

### Key Components

#### 1. CustomersTable

- Displays all customers with contact information
- Role-based access (admin/employee)
- Status management and filtering
- Loyalty tier display

#### 2. CustomerInteractions

- Complete interaction history
- Add new interactions (calls, emails, meetings, etc.)
- Priority and status management
- Follow-up scheduling

#### 3. CustomerStats

- Customer-specific analytics
- Loyalty program information
- Interaction breakdown by type
- Quick action buttons

## 🎯 **Customer Types**

### Individual Customers

- Personal contact information
- Individual loyalty accounts
- Personal interaction tracking

### Business Customers

- Company information
- Multiple contacts per company
- Business-specific interactions
- Corporate loyalty programs

## 📊 **Loyalty Program**

### Tiers

- **Bronze**: 0-999 points
- **Silver**: 1,000-4,999 points
- **Gold**: 5,000-9,999 points
- **Platinum**: 10,000+ points

### Features

- Points earning and redemption tracking
- Tier-based benefits
- Activity monitoring
- Lifetime statistics

## 🔍 **Search & Filtering**

### Customer Search

- By name, email, phone
- By company name
- By location (city, state, country)
- By customer type and status

### Advanced Filters

- Date ranges (created, last contact)
- Interaction types
- Loyalty tier
- Customer segments

## 📈 **Analytics & Reporting**

### CRM Statistics

- Total customers by type and status
- Recent interaction counts
- Pending follow-ups
- Customer acquisition trends

### Customer Analytics

- Interaction history
- Loyalty program participation
- Communication preferences
- Engagement metrics

## 🎨 **User Interface**

### Dashboard Layout

- **Header**: Search and quick actions
- **Stats Cards**: Key metrics at a glance
- **Filters**: Advanced search and filtering
- **Tabs**: Customers, Interactions, Analytics

### Responsive Design

- Mobile-friendly interface
- Tablet optimization
- Desktop full-featured view

## 🔐 **Security & Access Control**

### Role-Based Access

- **Admin**: Full CRM access
- **Employee**: Limited CRM access (no customer deletion)
- **User**: No CRM access

### Data Protection

- Secure customer data handling
- Audit trail for all changes
- Privacy-compliant data storage

## 🚀 **Getting Started**

### 1. Access CRM

Navigate to `/admin/crm` in the admin dashboard

### 2. Add Customers

- Click "Add Customer" button
- Fill in customer information
- Set customer type and status

### 3. Track Interactions

- Select a customer
- Go to "Interactions" tab
- Add new interactions
- Set priorities and follow-ups

### 4. View Analytics

- Select a customer
- Go to "Analytics" tab
- View customer statistics
- Monitor loyalty program status

## 🔧 **Configuration**

### Customer Types

- Individual: Personal customers
- Business: Corporate customers

### Interaction Types

- Call: Phone conversations
- Email: Email communications
- Meeting: In-person or virtual meetings
- Support: Customer support interactions
- Sale: Sales-related activities
- Follow-up: Scheduled follow-ups
- Other: Miscellaneous interactions

### Priority Levels

- Low: Routine interactions
- Medium: Standard priority
- High: Important interactions
- Urgent: Critical issues

## 📱 **Mobile Support**

The CRM system is fully responsive and works on:

- Desktop computers
- Tablets
- Mobile phones

## 🔄 **Integration**

### With Existing Systems

- **User Management**: Links customers to user accounts
- **Loyalty Program**: Integrated with points system
- **Order Management**: Customer order history
- **Product Catalog**: Customer preferences

### Future Enhancements

- Email integration
- Calendar synchronization
- SMS notifications
- Advanced reporting
- Customer segmentation automation

## 📚 **Best Practices**

### Customer Management

1. Keep customer information up-to-date
2. Regular interaction logging
3. Follow-up scheduling
4. Status management

### Data Quality

1. Consistent data entry
2. Regular data cleanup
3. Duplicate prevention
4. Data validation

### Security

1. Regular access reviews
2. Data backup procedures
3. Privacy compliance
4. Audit trail maintenance

This CRM system provides a comprehensive solution for managing customer relationships, similar to enterprise solutions like Odoo, while being tailored specifically for the FMBQ application's needs.
