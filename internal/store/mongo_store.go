package store

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/pastorenue/kflow/pkg/kflow"
)

const (
	collExecutions = "kflow_executions"
	collStates     = "kflow_states"
)

var _ Store = (*MongoStore)(nil)

// executionDoc is the BSON representation of an ExecutionRecord.
type executionDoc struct {
	ID        string    `bson:"_id"`
	Workflow  string    `bson:"workflow"`
	Status    string    `bson:"status"`
	Input     bson.M    `bson:"input"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

// stateDoc is the BSON representation of a StateRecord.
// _id is "<execID>:<stateName>:<attempt>".
type stateDoc struct {
	ID          string    `bson:"_id"`
	ExecutionID string    `bson:"execution_id"`
	StateName   string    `bson:"state_name"`
	Status      string    `bson:"status"`
	Input       bson.M    `bson:"input"`
	Output      bson.M    `bson:"output"`
	Error       string    `bson:"error"`
	Attempt     int       `bson:"attempt"`
	CreatedAt   time.Time `bson:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at"`
}

// MongoStore is the production implementation of Store backed by MongoDB.
type MongoStore struct {
	client *mongo.Client
	db     *mongo.Database
}

// NewMongoStore connects to MongoDB, ensures indexes, and returns a ready store.
func NewMongoStore(ctx context.Context, uri, dbName string) (*MongoStore, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("store: mongo connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("store: mongo ping: %w", err)
	}
	s := &MongoStore{
		client: client,
		db:     client.Database(dbName),
	}
	if err := s.EnsureIndexes(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

// EnsureIndexes creates required indexes if they do not already exist.
func (s *MongoStore) EnsureIndexes(ctx context.Context) error {
	execIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("executions_status_idx"),
		},
		{
			Keys:    bson.D{{Key: "workflow", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("executions_workflow_created_idx"),
		},
	}
	if _, err := s.db.Collection(collExecutions).Indexes().CreateMany(ctx, execIndexes); err != nil {
		return fmt.Errorf("store: ensure execution indexes: %w", err)
	}

	stateIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "execution_id", Value: 1}, {Key: "state_name", Value: 1}},
			Options: options.Index().SetName("states_exec_state_idx"),
		},
		{
			Keys:    bson.D{{Key: "execution_id", Value: 1}, {Key: "status", Value: 1}},
			Options: options.Index().SetName("states_exec_status_idx"),
		},
	}
	if _, err := s.db.Collection(collStates).Indexes().CreateMany(ctx, stateIndexes); err != nil {
		return fmt.Errorf("store: ensure state indexes: %w", err)
	}

	return nil
}

func (s *MongoStore) CreateExecution(ctx context.Context, record ExecutionRecord) error {
	now := time.Now()
	doc := executionDoc{
		ID:        record.ID,
		Workflow:  record.Workflow,
		Status:    string(StatusPending),
		Input:     inputToM(record.Input),
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := s.db.Collection(collExecutions).InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("store: execution %q already exists", record.ID)
		}
		return fmt.Errorf("store: create execution: %w", err)
	}
	return nil
}

func (s *MongoStore) GetExecution(ctx context.Context, execID string) (ExecutionRecord, error) {
	var doc executionDoc
	err := s.db.Collection(collExecutions).FindOne(ctx,
		bson.D{{Key: "_id", Value: execID}},
	).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return ExecutionRecord{}, ErrExecutionNotFound
		}
		return ExecutionRecord{}, fmt.Errorf("store: get execution: %w", err)
	}
	return ExecutionRecord{
		ID:        doc.ID,
		Workflow:  doc.Workflow,
		Status:    Status(doc.Status),
		Input:     mToInput(doc.Input),
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}, nil
}

func (s *MongoStore) UpdateExecution(ctx context.Context, execID string, status Status) error {
	result, err := s.db.Collection(collExecutions).UpdateOne(ctx,
		bson.D{{Key: "_id", Value: execID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "status", Value: string(status)},
			{Key: "updated_at", Value: time.Now()},
		}}},
	)
	if err != nil {
		return fmt.Errorf("store: update execution: %w", err)
	}
	if result.MatchedCount == 0 {
		return ErrExecutionNotFound
	}
	return nil
}

func (s *MongoStore) WriteAheadState(ctx context.Context, record StateRecord) error {
	coll := s.db.Collection(collStates)

	// Check for any existing terminal Completed record to enforce idempotency.
	var existing stateDoc
	err := coll.FindOne(ctx,
		bson.D{
			{Key: "execution_id", Value: record.ExecutionID},
			{Key: "state_name", Value: record.StateName},
			{Key: "status", Value: string(StatusCompleted)},
		},
	).Decode(&existing)
	if err != nil && err != mongo.ErrNoDocuments {
		return fmt.Errorf("store: write-ahead check: %w", err)
	}
	if err == nil {
		// A Completed document exists — idempotency guard.
		return ErrStateAlreadyTerminal
	}

	now := time.Now()
	doc := stateDoc{
		ID:          stateDocID(record.ExecutionID, record.StateName, record.Attempt),
		ExecutionID: record.ExecutionID,
		StateName:   record.StateName,
		Status:      string(StatusPending),
		Input:       inputToM(record.Input),
		Attempt:     record.Attempt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, insertErr := coll.InsertOne(ctx, doc)
	if insertErr != nil {
		if mongo.IsDuplicateKeyError(insertErr) {
			return ErrStateAlreadyTerminal
		}
		return fmt.Errorf("store: write-ahead insert: %w", insertErr)
	}
	return nil
}

func (s *MongoStore) MarkRunning(ctx context.Context, execID, stateName string) error {
	return s.updateLatestState(ctx, execID, stateName, bson.D{
		{Key: "status", Value: string(StatusRunning)},
		{Key: "updated_at", Value: time.Now()},
	})
}

func (s *MongoStore) CompleteState(ctx context.Context, execID, stateName string, output kflow.Output) error {
	return s.updateLatestState(ctx, execID, stateName, bson.D{
		{Key: "status", Value: string(StatusCompleted)},
		{Key: "output", Value: inputToM(output)},
		{Key: "updated_at", Value: time.Now()},
	})
}

func (s *MongoStore) FailState(ctx context.Context, execID, stateName string, errMsg string) error {
	return s.updateLatestState(ctx, execID, stateName, bson.D{
		{Key: "status", Value: string(StatusFailed)},
		{Key: "error", Value: errMsg},
		{Key: "updated_at", Value: time.Now()},
	})
}

func (s *MongoStore) GetStateOutput(ctx context.Context, execID, stateName string) (kflow.Output, error) {
	var doc stateDoc
	err := s.db.Collection(collStates).FindOne(ctx,
		bson.D{
			{Key: "execution_id", Value: execID},
			{Key: "state_name", Value: stateName},
		},
		options.FindOne().SetSort(bson.D{{Key: "attempt", Value: -1}}),
	).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrStateNotFound
		}
		return nil, fmt.Errorf("store: get state output: %w", err)
	}
	if doc.Status != string(StatusCompleted) {
		return nil, ErrStateNotCompleted
	}
	return mToInput(doc.Output), nil
}

// updateLatestState applies a $set update to the most recent attempt of a state.
func (s *MongoStore) updateLatestState(ctx context.Context, execID, stateName string, fields bson.D) error {
	// Find latest attempt first.
	var doc stateDoc
	err := s.db.Collection(collStates).FindOne(ctx,
		bson.D{
			{Key: "execution_id", Value: execID},
			{Key: "state_name", Value: stateName},
		},
		options.FindOne().SetSort(bson.D{{Key: "attempt", Value: -1}}),
	).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return ErrStateNotFound
		}
		return fmt.Errorf("store: find state for update: %w", err)
	}

	result, err := s.db.Collection(collStates).UpdateOne(ctx,
		bson.D{{Key: "_id", Value: doc.ID}},
		bson.D{{Key: "$set", Value: fields}},
	)
	if err != nil {
		return fmt.Errorf("store: update state: %w", err)
	}
	if result.MatchedCount == 0 {
		return ErrStateNotFound
	}
	return nil
}

// DropDatabase drops the database. Used only in tests.
func (s *MongoStore) DropDatabase(ctx context.Context) error {
	return s.db.Drop(ctx)
}

func stateDocID(execID, stateName string, attempt int) string {
	return fmt.Sprintf("%s:%s:%d", execID, stateName, attempt)
}

func inputToM(in map[string]any) bson.M {
	if in == nil {
		return bson.M{}
	}
	m := make(bson.M, len(in))
	for k, v := range in {
		m[k] = v
	}
	return m
}

func mToInput(m bson.M) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
