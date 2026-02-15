package ankylogo

import (
	"testing"
	"time"
)

/*
Test that the first denied event for an IP sets the score to 1
A brand new IP with no history should start at exactly 1
*/
func TestRiskScoreFirstEvent(t *testing.T) {
	engine := &RiskEngine{
		threshold: 5,
		decayRate: 30 * time.Minute,
	}

	event := RateLimitEvent{
		IP:        "192.168.1.1",
		Endpoint:  "GET /ping",
		Action:    "DENIED_WINDOW",
		Timestamp: time.Now().UnixNano(),
	}

	score := engine.processEvent(event)
	if score != 1 {
		t.Errorf("First event should set score to 1, got %d", score)
	}
}

/*
Test that multiple rapid events from the same IP accumulate correctly
5 events with no time for decay should produce a score of exactly 5
*/
func TestRiskScoreMultipleEvents(t *testing.T) {
	engine := &RiskEngine{
		threshold: 10,
		decayRate: 30 * time.Minute,
	}

	event := RateLimitEvent{
		IP:        "10.0.0.1",
		Endpoint:  "POST /login",
		Action:    "DENIED_BUCKET",
		Timestamp: time.Now().UnixNano(),
	}

	var lastScore int64
	for i := 0; i < 5; i++ {
		lastScore = engine.processEvent(event)
	}

	if lastScore != 5 {
		t.Errorf("After 5 rapid events score should be 5, got %d", lastScore)
	}
}

/*
Test that different IPs have completely isolated scores
Events from IP A should not affect IP B's score
*/
func TestRiskScoreIsolatedIPs(t *testing.T) {
	engine := &RiskEngine{
		threshold: 10,
		decayRate: 30 * time.Minute,
	}

	eventA := RateLimitEvent{IP: "1.1.1.1", Endpoint: "GET /ping", Action: "DENIED_WINDOW", Timestamp: time.Now().UnixNano()}
	eventB := RateLimitEvent{IP: "2.2.2.2", Endpoint: "GET /ping", Action: "DENIED_WINDOW", Timestamp: time.Now().UnixNano()}

	// 3 events for IP A
	for i := 0; i < 3; i++ {
		engine.processEvent(eventA)
	}

	// 1 event for IP B — should be 1, not 4
	scoreB := engine.processEvent(eventB)
	if scoreB != 1 {
		t.Errorf("IP B should have score 1 (isolated from A), got %d", scoreB)
	}

	// IP A should now be 4 (3 previous + 1 new)
	scoreA := engine.processEvent(eventA)
	if scoreA != 4 {
		t.Errorf("IP A should have score 4 after 4th event, got %d", scoreA)
	}
}

/*
Test that scores decay based on elapsed time
With a decay rate of 100ms, after 350ms a score of 5 should decay by 3
Then +1 for the new event = 3
*/
func TestRiskScoreDecay(t *testing.T) {
	engine := &RiskEngine{
		threshold: 10,
		decayRate: 100 * time.Millisecond,
	}

	event := RateLimitEvent{IP: "10.0.0.5", Endpoint: "GET /ping", Action: "DENIED_WINDOW", Timestamp: time.Now().UnixNano()}

	// Build up score to 5
	for i := 0; i < 5; i++ {
		engine.processEvent(event)
	}

	// Wait for ~3 decay intervals
	time.Sleep(350 * time.Millisecond)

	// Score was 5, decay 3, +1 = 3
	score := engine.processEvent(event)
	if score != 3 {
		t.Errorf("After 3 decay intervals, score should be 5-3+1=3, got %d", score)
	}
}

/*
Test that score decay floors at 0 and never goes negative
With a score of 2 and enough time for 10 decay intervals,
the score should floor at 0 then +1 for the new event = 1
*/
func TestRiskScoreDecayFloor(t *testing.T) {
	engine := &RiskEngine{
		threshold: 10,
		decayRate: 100 * time.Millisecond,
	}

	event := RateLimitEvent{IP: "172.16.0.1", Endpoint: "GET /search", Action: "DENIED_BUCKET", Timestamp: time.Now().UnixNano()}

	// Build up score to 2
	engine.processEvent(event)
	engine.processEvent(event)

	// Wait long enough that decay far exceeds current score
	time.Sleep(600 * time.Millisecond)

	// Score was 2, decay 6, floors at 0, +1 = 1
	score := engine.processEvent(event)
	if score != 1 {
		t.Errorf("After excessive decay, score should floor at 0 then +1 = 1, got %d", score)
	}
}

/*
Test threshold crossing detection
With a threshold of 3, events 1-3 should be at or below threshold
The 4th event should cross it
*/
func TestRiskScoreThresholdCrossing(t *testing.T) {
	engine := &RiskEngine{
		threshold: 3,
		decayRate: 30 * time.Minute,
	}

	event := RateLimitEvent{IP: "192.168.0.100", Endpoint: "POST /login", Action: "DENIED_WINDOW", Timestamp: time.Now().UnixNano()}

	// First 3 events should not exceed threshold
	for i := 0; i < 3; i++ {
		score := engine.processEvent(event)
		if score > engine.threshold {
			t.Errorf("Event %d should not exceed threshold of 3, score is %d", i+1, score)
		}
	}

	// 4th event should exceed threshold
	score := engine.processEvent(event)
	if score <= engine.threshold {
		t.Errorf("4th event should exceed threshold of 3, score is %d", score)
	}
}
