// internal/repository/mongostore.go
package repository

import (
	"context"
	"time"

	"todo/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoStore struct {
	client  *mongo.Client
	db      string
	coll    string
	timeout time.Duration
}

func NewMongoStore(uri, db, coll string) (*MongoStore, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	return &MongoStore{
		client:  client,
		db:      db,
		coll:    coll,
		timeout: 5 * time.Second,
	}, nil
}

type taskDoc struct {
	ID          model.ID        `bson:"_id"`
	Title       string          `bson:"title"`
	Description string          `bson:"description,omitempty"`
	Status      model.Status    `bson:"status"`
	Priority    model.Priority  `bson:"priority"`
	DueAt       *time.Time      `bson:"due_at,omitempty"`
	CreatedAt   time.Time       `bson:"created_at"`
	UpdatedAt   time.Time       `bson:"updated_at"`
	CompletedAt *time.Time      `bson:"completed_at,omitempty"`
}

func (d taskDoc) toDTO() model.TaskDTO {
	return model.TaskDTO{
		ID:          d.ID,
		Title:       d.Title,
		Description: d.Description,
		Status:      d.Status,
		Priority:    d.Priority,
		DueAt:       d.DueAt,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
		CompletedAt: d.CompletedAt,
	}
}

func dtoToDoc(t model.TaskDTO) taskDoc {
	return taskDoc{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Status:      t.Status,
		Priority:    t.Priority,
		DueAt:       t.DueAt,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		CompletedAt: t.CompletedAt,
	}
}

func (s *MongoStore) Load() ([]model.TaskDTO, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	col := s.client.Database(s.db).Collection(s.coll)
	cur, err := col.Find(ctx, bson.D{})
	if err != nil {
		// если коллекции нет — вернём пусто
		if err == mongo.ErrNilDocument {
			return []model.TaskDTO{}, nil
		}
		return nil, err
	}
	defer cur.Close(ctx)

	var items []model.TaskDTO
	for cur.Next(ctx) {
		var d taskDoc
		if err := cur.Decode(&d); err != nil {
			return nil, err
		}
		items = append(items, d.toDTO())
	}
	return items, cur.Err()
}

func (s *MongoStore) Save(items []model.TaskDTO) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	col := s.client.Database(s.db).Collection(s.coll)
	if _, err := col.DeleteMany(ctx, bson.D{}); err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}
	docs := make([]any, 0, len(items))
	for _, it := range items {
		docs = append(docs, dtoToDoc(it))
	}
	_, err := col.InsertMany(ctx, docs)
	return err
}