package models

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Snippet struct {
	Author     map[string]string  `bson:"Author"`
	IDStr      string             `bson:"idstr"`
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	Title      string             `bson:"title"`
	Content    string             `bson:"content"`
	Created    time.Time          `bson:"created"`
	Tag        string             `bson:"tag"`
	Favourited int                `bson:"favourited"`
}

type SnippetModel struct {
	Client *mongo.Client
}

func (m *SnippetModel) Insert(title, content, tag, username, userIDStr string) (primitive.ObjectID, error) {
	collection := m.Client.Database("snippetbox").Collection("snippets")
	snippet := Snippet{
		Author:  map[string]string{username: userIDStr},
		Title:   title,
		Content: content,
		Created: time.Now().UTC(),
		Tag:     tag,
	}
	result, err := collection.InsertOne(context.TODO(), snippet)
	if err != nil {
		return primitive.NilObjectID, err
	}
	collection.FindOne(context.TODO(), bson.M{"_id": result.InsertedID}).Decode(&snippet)
	ID := snippet.ID
	IDstr := ID.Hex()
	_, err = collection.UpdateOne(context.TODO(), bson.M{"_id": result.InsertedID}, bson.M{"$set": bson.M{"idstr": IDstr}})
	if err != nil {
		return primitive.NilObjectID, err
	}
	id := result.InsertedID.(primitive.ObjectID)

	collection = m.Client.Database("snippetbox").Collection("users")
	filter := bson.M{
		"idstr": userIDStr,
	}
	_, err = collection.UpdateOne(context.TODO(), filter, bson.M{"$push": bson.M{"created_snippets": snippet}})
	if err != nil {
		return primitive.NilObjectID, err
	}

	return id, nil
}

func (m *SnippetModel) Get(id primitive.ObjectID) (Snippet, error) {
	collection := m.Client.Database("snippetbox").Collection("snippets")
	filter := bson.M{
		"_id": id,
	}
	var snippet Snippet
	err := collection.FindOne(context.TODO(), filter).Decode(&snippet)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Snippet{}, ErrNoRecord
		} else {
			return Snippet{}, err
		}
	}
	return snippet, nil
}

func (m *SnippetModel) Latest() ([]Snippet, error) {
	collection := m.Client.Database("snippetbox").Collection("snippets")
	filter := bson.M{}
	options := options.Find().SetSort(bson.M{"_id": -1}).SetLimit(10)
	cur, err := collection.Find(context.TODO(), filter, options)
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.TODO())
	var snippets []Snippet
	for cur.Next(context.TODO()) {
		var snippet Snippet
		err := cur.Decode(&snippet)
		if err != nil {
			return nil, err
		}
		snippets = append(snippets, snippet)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return snippets, nil
}
