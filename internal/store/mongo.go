package store

import (
	"context"
	"errors"

	"github.com/ayush3160/kartly-e2e/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Docs is the document store (Mongo): carts, order notes and reviews.
type Docs struct {
	client *mongo.Client
	db     *mongo.Database
}

// NewDocs connects to Mongo.
func NewDocs(uri, name string) (*Docs, error) {
	c, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	return &Docs{client: c, db: c.Database(name)}, nil
}

// Cart returns a cart, or an empty one.
func (d *Docs) Cart(ctx context.Context, id, currency string) (domain.Cart, error) {
	var c domain.Cart
	err := d.db.Collection("carts").FindOne(ctx, bson.M{"_id": id}).Decode(&c)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.Cart{ID: id, Items: []domain.CartItem{}, Currency: currency}, nil
	}
	if c.Items == nil {
		c.Items = []domain.CartItem{}
	}
	return c, err
}

// SaveCart replaces a cart.
func (d *Docs) SaveCart(ctx context.Context, c domain.Cart) error {
	_, err := d.db.Collection("carts").ReplaceOne(ctx, bson.M{"_id": c.ID}, c, options.Replace().SetUpsert(true))
	return err
}

// DeleteCart removes a cart after checkout.
func (d *Docs) DeleteCart(ctx context.Context, id string) error {
	_, err := d.db.Collection("carts").DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// OrderNotes returns the support notes on an order.
func (d *Docs) OrderNotes(ctx context.Context, orderID string) ([]string, error) {
	cur, err := d.db.Collection("order_notes").Find(ctx, bson.M{"orderId": orderID},
		options.Find().SetSort(bson.D{{Key: "seq", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var docs []struct {
		Text string `bson:"text"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := []string{}
	for _, n := range docs {
		out = append(out, n.Text)
	}
	return out, nil
}

// Rating is a product's review summary.
type Rating struct {
	Average float64 `json:"average"`
	Count   int     `json:"count"`
}

// ProductRating aggregates a product's reviews.
func (d *Docs) ProductRating(ctx context.Context, productID string) (Rating, error) {
	cur, err := d.db.Collection("reviews").Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"productId": productID}}},
		{{Key: "$group", Value: bson.M{"_id": nil, "avg": bson.M{"$avg": "$stars"}, "n": bson.M{"$sum": 1}}}},
	})
	if err != nil {
		return Rating{}, err
	}
	var rows []struct {
		Avg float64 `bson:"avg"`
		N   int     `bson:"n"`
	}
	if err := cur.All(ctx, &rows); err != nil || len(rows) == 0 {
		return Rating{}, err
	}
	return Rating{Average: float64(int(rows[0].Avg*10)) / 10, Count: rows[0].N}, nil
}

// Close disconnects.
func (d *Docs) Close(ctx context.Context) { _ = d.client.Disconnect(ctx) }
