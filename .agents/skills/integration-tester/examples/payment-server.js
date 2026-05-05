/**
 * Simple Payment Server
 * Handles orders, payments, captures, and refunds
 */

const express = require('express');
const app = express();
const PORT = 3001;

app.use(express.json());

// In-memory storage
const orders = new Map();
const payments = new Map();
const refunds = new Map();

// Generate unique IDs
const generateId = (prefix) => `${prefix}_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

// Middleware: Simple auth
app.use((req, res, next) => {
  const authHeader = req.headers.authorization;
  if (!authHeader || !authHeader.startsWith('Basic ')) {
    return res.status(401).json({ error: 'Authentication required' });
  }
  next();
});

// Health check
app.get('/health', (req, res) => {
  res.json({ status: 'ok', service: 'payment-server' });
});

// Create Order
app.post('/api/orders', (req, res) => {
  const { amount, currency, customer_id } = req.body;

  if (!amount || amount <= 0) {
    return res.status(400).json({ error: 'Invalid amount' });
  }

  const order = {
    id: generateId('ord'),
    amount,
    currency: currency || 'INR',
    customer_id,
    status: 'created',
    created_at: new Date().toISOString()
  };

  orders.set(order.id, order);
  res.status(201).json(order);
});

// Get Order
app.get('/api/orders/:id', (req, res) => {
  const order = orders.get(req.params.id);
  if (!order) {
    return res.status(404).json({ error: 'Order not found' });
  }
  res.json(order);
});

// Create Payment
app.post('/api/payments', (req, res) => {
  const { order_id, amount, method } = req.body;

  const order = orders.get(order_id);
  if (!order) {
    return res.status(404).json({ error: 'Order not found' });
  }

  if (order.status === 'paid') {
    return res.status(400).json({ error: 'Order already paid' });
  }

  if (amount !== order.amount) {
    return res.status(400).json({ error: 'Payment amount must match order amount' });
  }

  const payment = {
    id: generateId('pay'),
    order_id,
    amount,
    method: method || 'card',
    status: 'authorized',
    captured: false,
    refunded_amount: 0,
    created_at: new Date().toISOString()
  };

  payments.set(payment.id, payment);
  res.status(201).json(payment);
});

// Get Payment
app.get('/api/payments/:id', (req, res) => {
  const payment = payments.get(req.params.id);
  if (!payment) {
    return res.status(404).json({ error: 'Payment not found' });
  }
  res.json(payment);
});

// Capture Payment
app.post('/api/payments/:id/capture', (req, res) => {
  const payment = payments.get(req.params.id);

  if (!payment) {
    return res.status(404).json({ error: 'Payment not found' });
  }

  if (payment.status !== 'authorized') {
    return res.status(400).json({ error: 'Payment cannot be captured' });
  }

  if (payment.captured) {
    return res.status(400).json({ error: 'Payment already captured' });
  }

  // Capture the payment
  payment.status = 'captured';
  payment.captured = true;
  payment.captured_at = new Date().toISOString();

  // Update order status
  const order = orders.get(payment.order_id);
  if (order) {
    order.status = 'paid';
    order.paid_at = new Date().toISOString();
  }

  payments.set(payment.id, payment);
  res.json(payment);
});

// Create Refund
app.post('/api/refunds', (req, res) => {
  const { payment_id, amount, reason } = req.body;

  const payment = payments.get(payment_id);
  if (!payment) {
    return res.status(404).json({ error: 'Payment not found' });
  }

  if (!payment.captured) {
    return res.status(400).json({ error: 'Cannot refund uncaptured payment' });
  }

  const refundableAmount = payment.amount - payment.refunded_amount;
  if (amount > refundableAmount) {
    return res.status(400).json({
      error: 'Refund amount exceeds refundable amount',
      refundable_amount: refundableAmount
    });
  }

  const refund = {
    id: generateId('rfnd'),
    payment_id,
    amount,
    reason,
    status: 'processed',
    created_at: new Date().toISOString()
  };

  // Update payment
  payment.refunded_amount += amount;
  if (payment.refunded_amount === payment.amount) {
    payment.status = 'refunded';
  } else {
    payment.status = 'partially_refunded';
  }

  refunds.set(refund.id, refund);
  payments.set(payment_id, payment);

  res.status(201).json(refund);
});

// Get Refund
app.get('/api/refunds/:id', (req, res) => {
  const refund = refunds.get(req.params.id);
  if (!refund) {
    return res.status(404).json({ error: 'Refund not found' });
  }
  res.json(refund);
});

// List all refunds for a payment
app.get('/api/payments/:id/refunds', (req, res) => {
  const paymentRefunds = Array.from(refunds.values())
    .filter(r => r.payment_id === req.params.id);

  res.json({
    count: paymentRefunds.length,
    items: paymentRefunds
  });
});

// Start server
app.listen(PORT, () => {
  console.log(`✓ Payment Server running on http://localhost:${PORT}`);
  console.log(`✓ Health check: http://localhost:${PORT}/health`);
  console.log(`\nEndpoints:`);
  console.log(`  POST   /api/orders`);
  console.log(`  GET    /api/orders/:id`);
  console.log(`  POST   /api/payments`);
  console.log(`  GET    /api/payments/:id`);
  console.log(`  POST   /api/payments/:id/capture`);
  console.log(`  POST   /api/refunds`);
  console.log(`  GET    /api/refunds/:id`);
  console.log(`  GET    /api/payments/:id/refunds`);
});
