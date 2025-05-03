package controllers

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Represents a simplified version of the MPD XML structure
type MPD struct {
	XMLName xml.Name `xml:"MPD"`
	Period  Period   `xml:"Period"`
}

type Period struct {
	AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
	SegmentTemplate SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentTemplate struct {
	SegmentTimeline SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
	S []Segment `xml:"S"`
}

type Segment struct {
	T int64 `xml:"t,attr"` // start time
	D int64 `xml:"d,attr"` // duration
	R int64 `xml:"r,attr"` // repeat count (optional)
}

// Start cleaner in background
func StartDashCleaner(dashDir string) {
	a := 0
	for {
		time.Sleep(10 * time.Second) // run every 10 seconds
		count := cleanOldSegments(dashDir)
		log.Printf("Cleaned %d old segments\n", count)
		if count == 0 {
			print("no segments to clean\n")
			a++
		}
		if a > 10 {
			print("no segments to clean for 10 times\n")
			break
		}
	}
}

// Now cleanOldSegments RETURNS an int
func cleanOldSegments(dashDir string) int {
	mpdPath := filepath.Join(dashDir, "stream.mpd")

	// Read MPD file
	data, err := os.ReadFile(mpdPath)
	if err != nil {
		return 0
	}

	var mpd MPD
	err = xml.Unmarshal(data, &mpd)
	if err != nil {
		return 0
	}

	// Build set of valid segments
	validSegments := make(map[string]struct{})

	for _, adaptationSet := range mpd.Period.AdaptationSets {
		for _, segment := range adaptationSet.SegmentTemplate.SegmentTimeline.S {
			segmentName := buildSegmentName(segment.T) // customize this if needed
			validSegments[segmentName] = struct{}{}
		}
	}

	// List all .m4s files
	files, err := os.ReadDir(dashDir)
	if err != nil {
		return 0
	}

	count := 0
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".m4s") {
			if _, ok := validSegments[file.Name()]; !ok {
				err := os.Remove(filepath.Join(dashDir, file.Name()))
				if err == nil {
					count++
					log.Printf("Deleted old segment: %s\n", file.Name())
				} else {
					log.Printf("Failed to delete segment: %s, error: %v\n", file.Name(), err)
				}
			}
		}
	}

	return count
}

// Build the segment filename based on timestamp or ID
func buildSegmentName(timestamp int64) string {
	return fmt.Sprintf("chunk-stream0-%d.m4s", timestamp)
}
