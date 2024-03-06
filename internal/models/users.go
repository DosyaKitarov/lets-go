package models

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type UserModelInterface interface {
	Insert(name, email, password string) error
	Authenticate(email, password string) (int, error)
	Exists(id int) (bool, error)
	Get(id int) (User, error)
}

type User struct {
	IDStr           string             `bson:"idstr"`
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	Name            string             `bson:"name"`
	Email           string             `bson:"email"`
	HashedPassword  string             `bson:"hashed_password"`
	Created         time.Time          `bson:"created"`
	Favourites      []Snippet          `bson:"favourites"`
	CreatedSnippets []Snippet          `bson:"created_snippets"`
}

type UserModel struct {
	Client *mongo.Client
}

func (m *UserModel) Insert(name, email, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	collection := m.Client.Database("snippetbox").Collection("users")
	user := bson.M{
		"name":             name,
		"email":            email,
		"hashed_password":  hashedPassword,
		"created":          time.Now().UTC(),
		"favourites":       []Snippet{},
		"created_snippets": []Snippet{},
	}

	result, err := collection.InsertOne(context.TODO(), user)
	if err != nil {
		if strings.Contains(err.Error(), "E11000 duplicate key error") {
			return ErrDuplicateEmail
		}
		return err
	}
	collection.FindOne(context.TODO(), bson.M{"_id": result.InsertedID}).Decode(&user)
	ID := user["_id"].(primitive.ObjectID)
	IDstr := ID.Hex()
	_, err = collection.UpdateOne(context.TODO(), bson.M{"_id": result.InsertedID}, bson.M{"$set": bson.M{"idstr": IDstr}})
	if err != nil {
		return err
	}
	return nil
}
func (m *UserModel) Authenticate(email, password string) (primitive.ObjectID, error, string) {
	var user User

	collection := m.Client.Database("snippetbox").Collection("users")
	filter := bson.M{"email": email}
	err := collection.FindOne(context.TODO(), filter).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return primitive.NilObjectID, ErrInvalidCredentials, ""
		}
		return primitive.NilObjectID, err, ""
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return primitive.NilObjectID, ErrInvalidCredentials, ""
		}
		return primitive.NilObjectID, err, ""
	}

	return user.ID, nil, user.Name
}

func (m *UserModel) Exists(id primitive.ObjectID) (bool, error) {
	collection := m.Client.Database("snippetbox").Collection("users")
	filter := bson.M{"_id": id}
	count, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *UserModel) Get(id primitive.ObjectID) (User, error) {
	var user User

	collection := m.Client.Database("snippetbox").Collection("users")
	filter := bson.M{"_id": id}
	err := collection.FindOne(context.TODO(), filter).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return User{}, ErrNoRecord
		}
		return User{}, err
	}

	return user, nil
}
func (m *UserModel) AddFavourites(SnippetID primitive.ObjectID, ID primitive.ObjectID) error {
	collection := m.Client.Database("snippetbox").Collection("users")

	var user User
	err := collection.FindOne(context.TODO(), bson.M{"_id": ID, "favourites._id": bson.M{"$in": []primitive.ObjectID{SnippetID}}}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
		} else {
			return err
		}
	} else {
		return errors.New("post is already in favourites")
	}

	collection = m.Client.Database("snippetbox").Collection("snippets")
	filter := bson.M{"_id": SnippetID}
	var Snippet Snippet
	_, err = collection.UpdateOne(context.TODO(), filter, bson.M{"$inc": bson.M{"favourited": 1}})
	if err != nil {
		return err
	}
	err = collection.FindOne(context.TODO(), filter).Decode(&Snippet)
	if err != nil {
		return err
	}

	collection = m.Client.Database("snippetbox").Collection("users")
	_, err = collection.UpdateOne(context.TODO(), bson.M{"_id": ID}, bson.M{"$push": bson.M{"favourites": Snippet}})
	if err != nil {
		return err
	}

	return nil
}
func (m *UserModel) RemoveFavourites(Snippet Snippet, SnippetID primitive.ObjectID, ID primitive.ObjectID) error {
	collection := m.Client.Database("snippetbox").Collection("snippets")
	filter := bson.M{"_id": SnippetID}
	_, err := collection.UpdateOne(context.TODO(), filter, bson.M{"$inc": bson.M{"favourited": -1}})
	if err != nil {
		return err
	}
	collection = m.Client.Database("snippetbox").Collection("users")
	_, err = collection.UpdateOne(context.TODO(), bson.M{"_id": ID}, bson.M{"$pull": bson.M{"favourites": Snippet}})
	if err != nil {
		return err
	}
	return nil
}
