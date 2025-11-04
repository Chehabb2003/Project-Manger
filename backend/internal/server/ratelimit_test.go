package server

import (
    "testing"
    "time"

    "golang.org/x/time/rate"
)

func TestMultiLimiter_Allow(t *testing.T) {
    // 2 events per second with burst 2
    ml := newMultiLimiter(rate.Limit(2), 2, time.Minute)
    key := "test"
    if !ml.allow(key) { t.Fatal("first allow should pass") }
    if !ml.allow(key) { t.Fatal("second allow should pass") }
    // third immediate call should be denied due to burst exhausted
    if ml.allow(key) { t.Fatal("third allow should be rate limited") }
}

