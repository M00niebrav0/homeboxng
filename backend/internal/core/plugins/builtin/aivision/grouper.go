package aivision

import (
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// ExifGroupingWindow is the max time difference in seconds for EXIF timestamps
	// to group images as the same item.
	ExifGroupingWindow = 60.0

	// MessageGroupingWindow is the max time difference in seconds for Discord/message
	// timestamps to group images as the same item (fallback when no EXIF).
	MessageGroupingWindow = 120.0
)

// ImageGrouper accumulates images into sessions and groups them by item.
type ImageGrouper struct {
	mu       sync.RWMutex
	sessions map[string]*VisionSession // userID -> active session
}

// NewImageGrouper creates a new image grouper.
func NewImageGrouper() *ImageGrouper {
	return &ImageGrouper{
		sessions: make(map[string]*VisionSession),
	}
}

// AddImages creates or updates a session for a user with new images.
// Returns the updated session.
func (g *ImageGrouper) AddImages(userID, channelID, messageID string, discordTS float64, images []*PendingImage) *VisionSession {
	g.mu.Lock()
	defer g.mu.Unlock()

	session, exists := g.sessions[userID]
	if !exists || session.State != "collecting" {
		session = &VisionSession{
			SessionID:   uuid.New().String()[:8],
			UserID:      userID,
			ChannelID:   channelID,
			CreatedAt:   time.Now(),
			LastImageAt: time.Now(),
			State:       "collecting",
		}
		g.sessions[userID] = session
	}

	// Extract EXIF timestamps for each image
	for _, img := range images {
		if img.ExifTS == 0 {
			exifTime := ExtractEXIFDateTime(img.ImageBytes)
			if !exifTime.IsZero() {
				img.ExifTS = float64(exifTime.Unix()) + float64(exifTime.Nanosecond())/1e9
			}
		}
		if img.DiscordTS == 0 {
			img.DiscordTS = discordTS
		}
		if img.MessageID == "" {
			img.MessageID = messageID
		}
	}

	session.Images = append(session.Images, images...)
	session.LastImageAt = time.Now()

	return session
}

// SetUserContext updates the accumulated user context for a session.
func (g *ImageGrouper) SetUserContext(userID, context string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if session, ok := g.sessions[userID]; ok {
		if session.UserContext != "" {
			session.UserContext += " " + context
		} else {
			session.UserContext = context
		}
	}
}

// ForceFlush retrieves and removes a user's session for immediate processing.
func (g *ImageGrouper) ForceFlush(userID string) *VisionSession {
	g.mu.Lock()
	defer g.mu.Unlock()

	session, ok := g.sessions[userID]
	if !ok {
		return nil
	}

	session.State = "processing"
	return session
}

// GetActiveSession returns a user's active collecting session, if any.
func (g *ImageGrouper) GetActiveSession(userID string) *VisionSession {
	g.mu.RLock()
	defer g.mu.RUnlock()

	session, ok := g.sessions[userID]
	if !ok || session.State != "collecting" {
		return nil
	}
	return session
}

// CompleteSession marks a session as done and removes it.
func (g *ImageGrouper) CompleteSession(userID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if session, ok := g.sessions[userID]; ok {
		session.State = "done"
		delete(g.sessions, userID)
	}
}

// GroupImages groups a session's images by item using heuristics.
// Returns groups where each group represents images of the same item.
//
// Grouping priorities:
// 1. Same message ID -> always same group
// 2. EXIF timestamps within ExifGroupingWindow -> same item
// 3. Discord timestamps within MessageGroupingWindow -> same item (fallback)
func (g *ImageGrouper) GroupImages(session *VisionSession) [][]*PendingImage {
	if len(session.Images) == 0 {
		return nil
	}

	if len(session.Images) == 1 {
		return [][]*PendingImage{session.Images}
	}

	// Start with each image in its own group
	groups := make([][]*PendingImage, 0)
	assigned := make([]bool, len(session.Images))

	// Pass 1: Group by message ID
	messageGroups := make(map[string][]*PendingImage)
	for i, img := range session.Images {
		if img.MessageID != "" {
			messageGroups[img.MessageID] = append(messageGroups[img.MessageID], img)
			assigned[i] = true
		}
	}
	for _, group := range messageGroups {
		groups = append(groups, group)
	}

	// Pass 2: Group unassigned images by EXIF proximity
	for i, img := range session.Images {
		if assigned[i] {
			continue
		}

		// Try to find an existing group to merge into
		merged := false
		for gi, group := range groups {
			for _, existing := range group {
				if img.ExifTS > 0 && existing.ExifTS > 0 {
					if math.Abs(img.ExifTS-existing.ExifTS) <= ExifGroupingWindow {
						groups[gi] = append(groups[gi], img)
						assigned[i] = true
						merged = true
						break
					}
				}
			}
			if merged {
				break
			}
		}

		if merged {
			continue
		}

		// Pass 3: Try Discord timestamp proximity
		for gi, group := range groups {
			for _, existing := range group {
				if math.Abs(img.BestTimestamp()-existing.BestTimestamp()) <= MessageGroupingWindow {
					groups[gi] = append(groups[gi], img)
					assigned[i] = true
					merged = true
					break
				}
			}
			if merged {
				break
			}
		}

		if !merged {
			// Create a new group
			groups = append(groups, []*PendingImage{img})
			assigned[i] = true
		}
	}

	return groups
}
