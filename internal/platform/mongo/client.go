package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Client struct {
	client *mongo.Client
}

// Connect creates a new MongoDB client and verifies the connection.
func Connect(uri string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	// Verify connectivity
	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	return &Client{client: client}, nil
}

// Database returns a reference to a specific DB without mutating state.
func (c *Client) Database(name string) *mongo.Database {
	return c.client.Database(name)
}

// Disconnect closes the MongoDB client.
func (c *Client) Disconnect() {
	_ = c.client.Disconnect(context.Background())
}
