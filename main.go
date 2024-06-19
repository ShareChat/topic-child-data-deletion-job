package main

import (
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/spanner"
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/api/iterator"
	sppb "google.golang.org/genproto/googleapis/spanner/v1"
	"log"
	"os"
	"strconv"
)

const (
	projectID = "sharechat-production"

	sourceInstanceID = "topic-cdc-test-instance"
	sourceDatabaseID = "sharechat"

	pubsubTopic                    = "topic-table-ids-pubsub"
	epochTimeStampOlderThanOneYear = 1686375840000
)

func main() {
	// read an environment variable
	// if the environment variable is not set, use a default value

	pageSizeInString := os.Getenv("PAGE_SIZE")

	if pageSizeInString == "" {
		pageSizeInString = "100"
	}

	pageSizeInt, _ := strconv.Atoi(pageSizeInString)

	maxGoroutinesInString := os.Getenv("MAX_GOROUTINES")

	if maxGoroutinesInString == "" {
		maxGoroutinesInString = "10"
	}

	maxGoroutines, _ := strconv.Atoi(maxGoroutinesInString)

	ctx := context.Background()

	// Create a new Spanner client for source database
	sourceSpannerClient, err := spanner.NewClient(ctx, fmt.Sprintf("projects/%s/instances/%s/databases/%s", projectID, sourceInstanceID, sourceDatabaseID))
	if err != nil {
		log.Fatalf("Failed to create Spanner client for source database: %v", err)
	}
	defer sourceSpannerClient.Close()

	// Create a Pub/Sub client
	pubsubClient, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer pubsubClient.Close()
	pubSubTopicToPublish := pubsubClient.Topic(pubsubTopic)

	var pageNumber int

	maxGoRoutinesChannel := make(chan struct{}, maxGoroutines)

	for {
		pageNumber = pageNumber + 1

		maxGoRoutinesChannel <- struct{}{}

		go func(pageNumber int) {

			defer func() {
				<-maxGoRoutinesChannel
			}()

			topicAndLastUpdatedAt, _ := getTopics(ctx, sourceSpannerClient, pageSizeInt, pageNumber)

			// send all the topics as a single JSON array
			topicsIdArray := make([]string, 0)

			for topicId, lastUpdatedAt := range topicAndLastUpdatedAt {
				if lastUpdatedAt < epochTimeStampOlderThanOneYear { // double check
					topicsIdArray = append(topicsIdArray, topicId)
				} else {
					// This shoudln't happen - so we panic
					panic("lastUpdatedAt is greater than 1686375840000")
				}
			}

			// finally publish
			publishToPubSub(ctx, pubSubTopicToPublish, topicsIdArray)

		}(pageNumber)

	}
}

func publishToPubSub(ctx context.Context, pubsubTopic *pubsub.Topic, topics []string) {
	// Publish the JSON data as a single message
	topicIdsAsJSON, err := json.Marshal(topics)
	result := pubsubTopic.Publish(ctx, &pubsub.Message{
		Data: topicIdsAsJSON,
	})

	// Wait for the result
	id, err := result.Get(ctx)
	if err != nil {
		log.Printf("Failed to publish message: %v", err)
	} else {
		log.Printf("Published message; msg ID: %v", id)
	}
}

func getTopics(ctx context.Context, client *spanner.Client, pageSize, pageNumber int) (map[string]float64, error) {
	stmt := spanner.Statement{
		SQL:    "SELECT topicId, lastUpdatedAt  FROM topic WHERE lastUpdatedAt < 1686375840000 AND (status IS NULL OR status='deleted' OR status='active') AND (giftingStatus IS NULL OR giftingStatus='DISABLED') LIMIT @limit OFFSET @offset",
		Params: map[string]interface{}{"limit": pageSize, "offset": ((pageNumber - 1) * pageSize)},
	}

	iter := client.Single().QueryWithOptions(ctx, stmt, spanner.QueryOptions{Priority: sppb.RequestOptions_PRIORITY_LOW})
	defer iter.Stop()

	var topicsLastUpdatedAt = make(map[string]float64)
	for {
		row, err := iter.Next()

		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		var topicId string
		var lastUpdatedAt float64
		if err := row.Columns(&topicId, &lastUpdatedAt); err != nil {
			return nil, err
		}
		topicsLastUpdatedAt[topicId] = lastUpdatedAt
	}

	return topicsLastUpdatedAt, nil
}
