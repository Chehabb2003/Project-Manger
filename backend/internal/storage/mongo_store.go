package storage

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoBlobStore struct {
	client *mongo.Client
	coll   *mongo.Collection
}

// ---------- BLOB STORE (ciphertext blobs) ----------

func NewMongoBlobStore(ctx context.Context, uri, dbName, collName string) (BlobStore, error) {
	if uri == "" {
		return nil, errors.New("mongo uri is empty")
	}
	cli, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	// Verify connection quickly
	pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := cli.Ping(pctx, nil); err != nil {
		_ = cli.Disconnect(ctx)
		return nil, err
	}

	coll := cli.Database(dbName).Collection(collName)

	// Ensure a unique index on _id
	_, _ = coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	return &mongoBlobStore{client: cli, coll: coll}, nil
}

func (m *mongoBlobStore) Put(ctx context.Context, id string, data []byte) error {
	if id == "" {
		return errors.New("empty id")
	}
	_, err := m.coll.UpdateByID(
		ctx,
		id,
		bson.M{
			"$set": bson.M{
				"data":      data,
				"updatedAt": time.Now(),
			},
			"$setOnInsert": bson.M{
				"createdAt": time.Now(),
			},
		},
		options.Update().SetUpsert(true),
	)
	return err
}

func (m *mongoBlobStore) Get(ctx context.Context, id string) ([]byte, error) {
	if id == "" {
		return nil, errors.New("empty id")
	}
	var doc struct {
		Data []byte `bson:"data"`
	}
	err := m.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, errors.New("not found")
	}
	return doc.Data, err
}

func (m *mongoBlobStore) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("empty id")
	}
	_, err := m.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (m *mongoBlobStore) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}

// ---------- META STORE (ItemMeta index) ----------

// Local copy to avoid import cycles with the vault package.
type ItemMeta struct {
	ID      string `bson:"id" json:"id"`
	Type    string `bson:"type" json:"type"`
	Created int64  `bson:"created" json:"created"`
	Updated int64  `bson:"updated" json:"updated"`
	Version int    `bson:"version" json:"version"`
}

type MetaStore interface {
	PutMeta(ctx context.Context, meta ItemMeta) error
	ListMeta(ctx context.Context, filter map[string]interface{}) ([]ItemMeta, error)
}

type MongoMetaStore struct {
	client *mongo.Client
	coll   *mongo.Collection
}

func NewMongoMetaStore(ctx context.Context, uri, dbName, collName string) (*MongoMetaStore, error) {
	if uri == "" {
		return nil, errors.New("mongo uri is empty")
	}
	cli, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	// Verify connection quickly
	pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := cli.Ping(pctx, nil); err != nil {
		_ = cli.Disconnect(ctx)
		return nil, err
	}

	coll := cli.Database(dbName).Collection(collName)

	// Helpful index on id (not _id) since we store item id under "id"
	_, _ = coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	return &MongoMetaStore{client: cli, coll: coll}, nil
}

func (m *MongoMetaStore) PutMeta(ctx context.Context, meta ItemMeta) error {
	if meta.ID == "" {
		return errors.New("empty meta.id")
	}
	// Upsert by logical id (not Mongo _id) so re-adds update metadata
	_, err := m.coll.UpdateOne(
		ctx,
		bson.M{"id": meta.ID},
		bson.M{
			"$set": bson.M{
				"type":    meta.Type,
				"created": meta.Created,
				"updated": meta.Updated,
				"version": meta.Version,
			},
			"$setOnInsert": bson.M{
				"createdAt": time.Now(),
			},
		},
		options.Update().SetUpsert(true),
	)
	return err
}

func (m *MongoMetaStore) ListMeta(ctx context.Context, filter map[string]interface{}) ([]ItemMeta, error) {
	cur, err := m.coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var results []ItemMeta
	for cur.Next(ctx) {
		var im ItemMeta
		if err := cur.Decode(&im); err == nil {
			results = append(results, im)
		}
	}
	return results, cur.Err()
}
