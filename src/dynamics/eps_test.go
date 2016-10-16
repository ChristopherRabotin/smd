package dynamics

import (
	"testing"
	"time"
)

func TestUnlimitedEPS(t *testing.T) {
	eps := NewUnlimitedEPS()
	if err := eps.Drain(0, 0, time.Now()); err != nil {
		t.Fatalf("draining EPS fails: %s\n", err)
	}
}

func TestTimedEPS(t *testing.T) {
	eps := NewTimedEPS(time.Duration(1)*time.Minute, time.Duration(2)*time.Minute)
	initTime := time.Now()
	if err := eps.Drain(0, 0, initTime); err != nil {
		t.Fatalf("draining fresh EPS fails: %s\n", err)
	}
	if err := eps.Drain(0, 0, initTime.Add(time.Duration(1)*time.Minute)); err != nil {
		t.Fatalf("draining EPS before empty fails: %s\n", err)
	}
	if err := eps.Drain(0, 0, initTime.Add(time.Duration(2)*time.Minute)); err == nil {
		t.Fatal("draining EPS at empty time does not fail\n")
	}
	if err := eps.Drain(0, 0, initTime.Add(time.Duration(90)*time.Second)); err == nil {
		t.Fatal("draining EPS while charging does not fail\n")
	}
	if err := eps.Drain(0, 0, initTime.Add(time.Duration(3)*time.Minute)); err != nil {
		t.Fatalf("draining EPS after charging fails: %s\n", err)
	}
}
