// Package mongodb implements the driven persistence port
// (application.UserRepository) with the official MongoDB Go driver.
package mongodb

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/oat431/7-solution-interview/internal/application"
	"github.com/oat431/7-solution-interview/internal/domain"
)

const (
	usersCollection = "users"
	emailIndexName  = "ux_users_email"
)

// userDoc is the persistence shape; BSON mapping lives at the adapter boundary.
type userDoc struct {
	ID           bson.ObjectID `bson:"_id,omitempty"`
	Name         string        `bson:"name"`
	Email        string        `bson:"email"`
	PasswordHash string        `bson:"password_hash"`
	CreatedAt    time.Time     `bson:"created_at"`
}

// UserRepository is the MongoDB adapter for application.UserRepository.
type UserRepository struct {
	coll *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{coll: db.Collection(usersCollection)}
}

// EnsureIndexes creates the unique email index. Idempotent: recreating the
// same name+spec is a no-op, so it is safe to call on every startup (A12).
func (r *UserRepository) EnsureIndexes(ctx context.Context) error {
	idx := mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true).SetName(emailIndexName),
	}
	_, err := r.coll.Indexes().CreateOne(ctx, idx)
	return err
}

func (r *UserRepository) Create(ctx context.Context, rec application.CreateUserRecord) (domain.User, error) {
	doc := userDoc{
		Name:         rec.Name,
		Email:        rec.Email,
		PasswordHash: rec.PasswordHash,
		CreatedAt:    time.Now().UTC(),
	}
	res, err := r.coll.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.User{}, domain.ErrEmailExists
		}
		return domain.User{}, err
	}

	id, ok := res.InsertedID.(bson.ObjectID)
	if !ok {
		return domain.User{}, errors.New("unexpected inserted id type")
	}
	return domain.User{
		ID:        id.Hex(),
		Name:      doc.Name,
		Email:     doc.Email,
		CreatedAt: doc.CreatedAt,
	}, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (domain.User, error) {
	oid, err := parseID(id)
	if err != nil {
		return domain.User{}, domain.ErrInvalidID
	}
	return r.getByObjectID(ctx, oid)
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (application.StoredUser, error) {
	var doc userDoc
	if err := r.coll.FindOne(ctx, bson.M{"email": email}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return application.StoredUser{}, domain.ErrNotFound
		}
		return application.StoredUser{}, err
	}
	return application.StoredUser{User: doc.toUser(), PasswordHash: doc.PasswordHash}, nil
}

func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	cursor, err := r.coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	users := make([]domain.User, 0)
	for cursor.Next(ctx) {
		var doc userDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		users = append(users, doc.toUser())
	}
	return users, cursor.Err()
}

func (r *UserRepository) Update(ctx context.Context, id string, in application.UpdateUserInput) (domain.User, error) {
	oid, err := parseID(id)
	if err != nil {
		return domain.User{}, domain.ErrInvalidID
	}

	set := bson.M{}
	if in.Name != nil {
		set["name"] = *in.Name
	}
	if in.Email != nil {
		set["email"] = *in.Email
	}

	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": oid}, bson.M{"$set": set})
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.User{}, domain.ErrEmailExists
		}
		return domain.User{}, err
	}
	if res.MatchedCount == 0 {
		return domain.User{}, domain.ErrNotFound
	}
	return r.getByObjectID(ctx, oid)
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	oid, err := parseID(id)
	if err != nil {
		return domain.ErrInvalidID
	}

	res, err := r.coll.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	return r.coll.CountDocuments(ctx, bson.M{})
}

func parseID(id string) (bson.ObjectID, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return bson.NilObjectID, err
	}
	return oid, nil
}

func (r *UserRepository) getByObjectID(ctx context.Context, oid bson.ObjectID) (domain.User, error) {
	var doc userDoc
	if err := r.coll.FindOne(ctx, bson.M{"_id": oid}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, err
	}
	return doc.toUser(), nil
}

func (d userDoc) toUser() domain.User {
	return domain.User{
		ID:        d.ID.Hex(),
		Name:      d.Name,
		Email:     d.Email,
		CreatedAt: d.CreatedAt,
	}
}
