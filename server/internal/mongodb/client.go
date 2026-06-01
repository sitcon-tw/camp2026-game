package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const (
	connectTimeout    = 10 * time.Second
	disconnectTimeout = 5 * time.Second
)

func NewClient(ctx context.Context, uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri).SetConnectTimeout(connectTimeout))
	if err != nil {
		return nil, fmt.Errorf("connect mongodb: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), disconnectTimeout)
		defer disconnectCancel()
		_ = client.Disconnect(disconnectCtx)

		return nil, fmt.Errorf("ping mongodb: %w", err)
	}

	return client, nil
}
