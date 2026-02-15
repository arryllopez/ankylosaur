package ankylogo

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

type RiskScore struct {
	score       int64
	lastUpdated time.Time
	mu          sync.Mutex
}

type ThresholdNotifier interface {
	Notify(ip string, score int64)
}

type RiskEngine struct {
	client      *kgo.Client
	ipScores    sync.Map
	threshold   int64
	topic       string
	decayRate   time.Duration
	OnThreshold ThresholdNotifier
}

func NewRiskEngine(client *kgo.Client, threshold int64, topic string, decayRate time.Duration) *RiskEngine {
	return &RiskEngine{
		client:    client,
		threshold: threshold,
		topic:     topic,
		decayRate: decayRate,
	}
}

func NewRiskScore(score int64, lastUpdated time.Time) *RiskScore {
	return &RiskScore{
		score:       score,
		lastUpdated: lastUpdated,
	}
}

// GetScore returns the current effective risk score for an IP,
// applying time-based decay without modifying stored state
func (r *RiskEngine) GetScore(ip string) int64 {
	val, ok := r.ipScores.Load(ip)
	if !ok {
		return 0
	}
	riskScore := val.(*RiskScore)
	riskScore.mu.Lock()
	defer riskScore.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(riskScore.lastUpdated)
	intervals := int64(elapsed / r.decayRate)
	current := riskScore.score - intervals
	if current < 0 {
		current = 0
	}
	return current
}

// This function takes a the failed events for a specific ip and increments its risk score
// for each failed attempt, if no failed attempts happen over a period of time, an interval system
// is in place, so for example if interval was 30 minutes then if no failed api calls happen within 2 hours
// the specific ip's risk score gets deducted by 4 points since there are 120 minutes in 2 hours and
// 120 / 30 =  4
func (r *RiskEngine) processEvent(event RateLimitEvent) int64 {
	// bump the score for the ip for each denied event
	newScore := &RiskScore{lastUpdated: time.Now()}
	score, _ := r.ipScores.LoadOrStore(event.IP, newScore)
	riskScore := score.(*RiskScore)
	riskScore.mu.Lock()
	now := time.Now()
	elapsed := now.Sub(riskScore.lastUpdated)
	intervals := int64(elapsed / r.decayRate)
	riskScore.score -= intervals
	if riskScore.score < 0 {
		riskScore.score = 0
	}
	riskScore.score += 1
	riskScore.lastUpdated = now
	currentScore := riskScore.score
	riskScore.mu.Unlock()
	return currentScore
}

func (r *RiskEngine) EventReader(ctx context.Context) {
	for {
		//poll fetches, this blocks until records do arrive
		fetches := r.client.PollFetches(ctx)

		//if case for cancelled context
		if ctx.Err() != nil {
			fmt.Println("context cancelled")
			return
		}

		//errors while fetching
		if errs := fetches.Errors(); len(errs) > 0 {
			fmt.Printf("Errors while fetching: %v\n", errs)
			continue
		}

		// populating a new instance of ratelimitevent by unmarshalling the record
		fetches.EachRecord(func(record *kgo.Record) {
			var event RateLimitEvent
			err := json.Unmarshal(record.Value, &event)
			if err != nil {
				return
			}
			currentScore := r.processEvent(event)

			// check if current score is above the threshold
			if currentScore > r.threshold && r.OnThreshold != nil {
				r.OnThreshold.Notify(event.IP, currentScore)
			}
		})

		// when client closes end the loop
		if fetches.IsClientClosed() {
			return
		}

		fmt.Println("Fetched a batch of records...")
	}
}
