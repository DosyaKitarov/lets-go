package models

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Commentary struct {
	Author  map[string]string `bson:"Author"`
	Content string            `bson:"content"`
	Created time.Time         `bson:"created"`
}

type CommentaryModel struct {
	Client *mongo.Client
}

func (c *CommentaryModel) AddComentary(ID primitive.ObjectID, Author map[string]string, Content string) error {
	collection := c.Client.Database("snippetbox").Collection("snippets")
	Commentary := Commentary{
		Author:  Author,
		Content: Content,
		Created: time.Now().UTC(),
	}
	filter := bson.M{
		"_id": ID,
	}
	result, err := collection.UpdateOne(context.TODO(), filter, bson.M{"$push": bson.M{"commentaries": Commentary}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrNoRecord
	}
	return nil
}
