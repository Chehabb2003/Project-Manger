package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var ErrUserNotFound = errors.New("user not found")

type MongoUserStore struct {
	cli  *mongo.Client
	coll *mongo.Collection
}

func NewMongoUserStore(ctx context.Context, uri, db, coll string) (*MongoUserStore, error) {
	opts := options.Client().ApplyURI(uri)
	cli, err := mongo.NewClient(opts)
	if err != nil {
		return nil, err
	}
	dialCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := cli.Connect(dialCtx); err != nil {
		return nil, err
	}
	// optional ping
	_ = cli.Ping(dialCtx, readpref.Primary())

	c := cli.Database(db).Collection(coll)

	// Ensure unique indexes on username and email
	_, _ = c.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	_, _ = c.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	return &MongoUserStore{cli: cli, coll: c}, nil
}

// Add inserts a new user. Returns an error if username already exists.
func (s *MongoUserStore) Add(u *User) error {
	email := strings.ToLower(strings.TrimSpace(u.Email))
	u.Email = email
	doc := bson.M{
		"username":    u.Username,
		"email":       email,
		"pass_hash":   u.PassHash,
		"roles":       u.Roles, // []Role is []string under the hood
		"totp_secret": strings.TrimSpace(u.TOTPSecret),
	}
	_, err := s.coll.InsertOne(context.Background(), doc)
	if wex, ok := err.(mongo.WriteException); ok {
		for _, we := range wex.WriteErrors {
			if we.Code == 11000 { // duplicate key
				return errors.New("username or email already exists")
			}
		}
	}
	return err
}

// FindByUsername loads a user by username.
func (s *MongoUserStore) FindByUsername(username string) (*User, error) {
	return s.findOne(bson.M{"username": username})
}

// FindByEmail loads a user by email.
func (s *MongoUserStore) FindByEmail(email string) (*User, error) {
	return s.findOne(bson.M{"email": strings.ToLower(strings.TrimSpace(email))})
}

func (s *MongoUserStore) findOne(filter interface{}) (*User, error) {
	var doc struct {
		Username   string `bson:"username"`
		Email      string `bson:"email"`
		PassHash   string `bson:"pass_hash"`
		Roles      []Role `bson:"roles"`
		TOTPSecret string `bson:"totp_secret"`
	}
	err := s.coll.FindOne(context.Background(), filter).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &User{
		Username:   doc.Username,
		Email:      doc.Email,
		PassHash:   doc.PassHash,
		Roles:      doc.Roles,
		TOTPSecret: doc.TOTPSecret,
	}, nil
}

// UpdatePassword replaces the stored password hash for a user.
func (s *MongoUserStore) UpdatePassword(username, newHash string) error {
	res, err := s.coll.UpdateOne(
		context.Background(),
		bson.M{"username": username},
		bson.M{"$set": bson.M{"pass_hash": newHash}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrUserNotFound
	}
	return nil
}
