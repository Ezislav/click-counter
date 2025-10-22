package storage

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type StatItem struct {
	TS    time.Time
	Count int64
}

type AllStatItem struct {
	BannerID string
	TS       time.Time
	Count    int64
}

type Repository interface {
	Inc(ctx context.Context, bannerID string, tsMinute time.Time, delta int64) error
	Range(ctx context.Context, bannerID string, from, to time.Time) ([]StatItem, error)
	RangeAll(ctx context.Context, from, to time.Time) ([]AllStatItem, error)
	Close(ctx context.Context) error
}

type mongoRepo struct {
	cli  *mongo.Client
	coll *mongo.Collection
}

func NewMongoRepo(ctx context.Context, uri, db, coll string) (Repository, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}
	collection := client.Database(db).Collection(coll)

	idx := mongo.IndexModel{
		Keys:    bson.D{{Key: "bannerId", Value: 1}, {Key: "tsMinute", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	if _, err := collection.Indexes().CreateOne(ctx, idx); err != nil {
		return nil, fmt.Errorf("create index: %w", err)
	}
	return &mongoRepo{cli: client, coll: collection}, nil
}

func (m *mongoRepo) docID(bannerID string, ts time.Time) string {
	return fmt.Sprintf("%s_%s", bannerID, ts.UTC().Format(time.RFC3339))
}

func (m *mongoRepo) Inc(ctx context.Context, bannerID string, tsMinute time.Time, delta int64) error {
	filter := bson.M{"_id": m.docID(bannerID, tsMinute)}
	update := bson.M{
		"$setOnInsert": bson.M{"bannerId": bannerID, "tsMinute": tsMinute},
		"$inc":         bson.M{"count": delta},
	}
	opts := options.Update().SetUpsert(true)
	_, err := m.coll.UpdateOne(ctx, filter, update, opts)
	return err
}

func (m *mongoRepo) Range(ctx context.Context, bannerID string, from, to time.Time) ([]StatItem, error) {

	filter := bson.M{"bannerId": bannerID, "tsMinute": bson.M{"$gte": from, "$lt": to}}
	opts := options.Find().SetSort(bson.D{{Key: "tsMinute", Value: 1}})
	cur, err := m.coll.Find(ctx, filter, opts)

	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	res := make([]StatItem, 0, 64)
	for cur.Next(ctx) {
		var d struct {
			TSMinute time.Time `bson:"tsMinute"`
			Count    int64     `bson:"count"`
		}
		if err := cur.Decode(&d); err != nil {
			return nil, err
		}
		res = append(res, StatItem{TS: d.TSMinute.UTC(), Count: d.Count})
	}
	return res, cur.Err()
}

func (m *mongoRepo) RangeAll(ctx context.Context, from, to time.Time) ([]AllStatItem, error) {
	filter := bson.M{
		"tsMinute": bson.M{"$gte": from, "$lt": to},
	}
	opts := options.Find().SetSort(bson.D{
		{Key: "bannerId", Value: 1},
		{Key: "tsMinute", Value: 1},
	})

	cur, err := m.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var res []AllStatItem
	for cur.Next(ctx) {
		var d struct {
			BannerID string    `bson:"bannerId"`
			TSMinute time.Time `bson:"tsMinute"`
			Count    int64     `bson:"count"`
		}
		if err := cur.Decode(&d); err != nil {
			return nil, err
		}
		res = append(res, AllStatItem{
			BannerID: d.BannerID,
			TS:       d.TSMinute.UTC(),
			Count:    d.Count,
		})
	}

	return res, cur.Err()
}

func (m *mongoRepo) Close(ctx context.Context) error {
	return m.cli.Disconnect(ctx)
}
