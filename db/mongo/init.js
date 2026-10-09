// Kartly documents: carts, order notes, reviews.
const kdb = db.getSiblingDB("kartly");

kdb.carts.insertMany([
  { _id: "cart-u-103", customerId: "u-103", currency: "INR", items: [
    { productId: "p-1001", sku: "SKU-TEE-BLK-M", name: "Classic Tee, Black, M", quantity: 2, unitPrice: { amount: 79900, currency: "INR" } },
  ] },
  { _id: "cart-u-104", customerId: "u-104", currency: "INR", items: [
    { productId: "p-2006", sku: "SKU-LOW-STOCK", name: "Limited Edition Keyboard", quantity: 3, unitPrice: { amount: 1499900, currency: "INR" } },
  ] },
  { _id: "guest-7f3a", currency: "INR", items: [
    { productId: "p-3001", sku: "SKU-MUG-CER", name: "Ceramic Mug, 350ml", quantity: 2, unitPrice: { amount: 49900, currency: "INR" } },
  ] },
]);

kdb.order_notes.insertMany([
  { orderId: "ord_seed_0002", seq: 1, text: "Customer asked to deliver after 6pm." },
  { orderId: "ord_seed_0002", seq: 2, text: "Courier confirmed evening slot." },
  { orderId: "ord_seed_0006", seq: 1, text: "High-value order: verified by phone." },
  { orderId: "ord_seed_0011", seq: 1, text: "Address updated by support before dispatch." },
]);

kdb.reviews.insertMany([
  { productId: "p-1001", stars: 5 }, { productId: "p-1001", stars: 4 }, { productId: "p-1001", stars: 4 },
  { productId: "p-2001", stars: 5 }, { productId: "p-2001", stars: 3 },
  { productId: "p-2002", stars: 4 },
  { productId: "p-3002", stars: 5 }, { productId: "p-3002", stars: 5 },
  { productId: "p-4002", stars: 5 }, { productId: "p-4002", stars: 5 }, { productId: "p-4002", stars: 4 },
]);
