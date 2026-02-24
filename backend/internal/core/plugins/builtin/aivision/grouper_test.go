package aivision

import (
	"testing"
)

func TestNewImageGrouper(t *testing.T) {
	g := NewImageGrouper()
	if g == nil {
		t.Fatal("expected non-nil grouper")
	}
	if g.sessions == nil {
		t.Fatal("expected initialized sessions map")
	}
}

func TestImageGrouper_AddImages_CreatesSession(t *testing.T) {
	g := NewImageGrouper()

	images := []*PendingImage{
		{Filename: "IMG_001.JPG", MimeType: "image/jpeg", ImageBytes: []byte{0xFF, 0xD8}},
	}

	session := g.AddImages("user1", "chan1", "msg1", 1000.0, images)
	if session == nil {
		t.Fatal("expected session to be created")
	}
	if session.UserID != "user1" {
		t.Errorf("UserID = %q, want %q", session.UserID, "user1")
	}
	if session.ChannelID != "chan1" {
		t.Errorf("ChannelID = %q, want %q", session.ChannelID, "chan1")
	}
	if session.State != "collecting" {
		t.Errorf("State = %q, want %q", session.State, "collecting")
	}
	if len(session.Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(session.Images))
	}
	if session.Images[0].MessageID != "msg1" {
		t.Errorf("MessageID = %q, want %q", session.Images[0].MessageID, "msg1")
	}
}

func TestImageGrouper_AddImages_AppendsToExisting(t *testing.T) {
	g := NewImageGrouper()

	batch1 := []*PendingImage{
		{Filename: "IMG_001.JPG", ImageBytes: []byte{0xFF, 0xD8}},
	}
	batch2 := []*PendingImage{
		{Filename: "IMG_002.JPG", ImageBytes: []byte{0xFF, 0xD8}},
		{Filename: "IMG_003.JPG", ImageBytes: []byte{0xFF, 0xD8}},
	}

	g.AddImages("user1", "chan1", "msg1", 1000.0, batch1)
	session := g.AddImages("user1", "chan1", "msg2", 1001.0, batch2)

	if len(session.Images) != 3 {
		t.Errorf("expected 3 images, got %d", len(session.Images))
	}
}

func TestImageGrouper_AddImages_SetsDiscordTS(t *testing.T) {
	g := NewImageGrouper()

	images := []*PendingImage{
		{Filename: "test.jpg", ImageBytes: []byte{0xFF, 0xD8}},
	}

	session := g.AddImages("user1", "chan1", "msg1", 12345.678, images)
	if session.Images[0].DiscordTS != 12345.678 {
		t.Errorf("DiscordTS = %f, want 12345.678", session.Images[0].DiscordTS)
	}
}

func TestImageGrouper_AddImages_PreservesExistingDiscordTS(t *testing.T) {
	g := NewImageGrouper()

	images := []*PendingImage{
		{Filename: "test.jpg", DiscordTS: 99999.0, ImageBytes: []byte{0xFF, 0xD8}},
	}

	session := g.AddImages("user1", "chan1", "msg1", 12345.678, images)
	if session.Images[0].DiscordTS != 99999.0 {
		t.Errorf("DiscordTS = %f, want 99999.0 (should preserve existing)", session.Images[0].DiscordTS)
	}
}

func TestImageGrouper_GetActiveSession(t *testing.T) {
	g := NewImageGrouper()

	// No session yet
	if s := g.GetActiveSession("user1"); s != nil {
		t.Error("expected nil for non-existent user")
	}

	// Create session
	g.AddImages("user1", "chan1", "msg1", 1000.0, []*PendingImage{
		{Filename: "test.jpg", ImageBytes: []byte{0xFF, 0xD8}},
	})

	s := g.GetActiveSession("user1")
	if s == nil {
		t.Fatal("expected active session")
	}
	if s.UserID != "user1" {
		t.Errorf("UserID = %q", s.UserID)
	}
}

func TestImageGrouper_SetUserContext(t *testing.T) {
	g := NewImageGrouper()

	// Setting context for non-existent user is a no-op
	g.SetUserContext("nouser", "context")

	g.AddImages("user1", "chan1", "msg1", 1000.0, []*PendingImage{
		{Filename: "test.jpg", ImageBytes: []byte{0xFF, 0xD8}},
	})

	g.SetUserContext("user1", "server rack items")
	s := g.GetActiveSession("user1")
	if s.UserContext != "server rack items" {
		t.Errorf("UserContext = %q", s.UserContext)
	}

	// Appending context
	g.SetUserContext("user1", "Supermicro stuff")
	s = g.GetActiveSession("user1")
	if s.UserContext != "server rack items Supermicro stuff" {
		t.Errorf("UserContext = %q, want concatenated", s.UserContext)
	}
}

func TestImageGrouper_ForceFlush(t *testing.T) {
	g := NewImageGrouper()

	// Flush non-existent
	if s := g.ForceFlush("nouser"); s != nil {
		t.Error("expected nil for non-existent user")
	}

	g.AddImages("user1", "chan1", "msg1", 1000.0, []*PendingImage{
		{Filename: "test.jpg", ImageBytes: []byte{0xFF, 0xD8}},
	})

	s := g.ForceFlush("user1")
	if s == nil {
		t.Fatal("expected session from flush")
	}
	if s.State != "processing" {
		t.Errorf("State = %q, want %q", s.State, "processing")
	}

	// After flush, new AddImages should create a new session (state != "collecting")
	newSession := g.AddImages("user1", "chan1", "msg2", 2000.0, []*PendingImage{
		{Filename: "new.jpg", ImageBytes: []byte{0xFF, 0xD8}},
	})
	if newSession.SessionID == s.SessionID {
		t.Error("expected new session after flush")
	}
}

func TestImageGrouper_CompleteSession(t *testing.T) {
	g := NewImageGrouper()

	g.AddImages("user1", "chan1", "msg1", 1000.0, []*PendingImage{
		{Filename: "test.jpg", ImageBytes: []byte{0xFF, 0xD8}},
	})

	g.CompleteSession("user1")

	if s := g.GetActiveSession("user1"); s != nil {
		t.Error("expected nil after complete")
	}

	// Complete non-existent is a no-op
	g.CompleteSession("nouser")
}

func TestImageGrouper_GroupImages_Empty(t *testing.T) {
	g := NewImageGrouper()
	session := &VisionSession{}
	groups := g.GroupImages(session)
	if groups != nil {
		t.Errorf("expected nil for empty session, got %v", groups)
	}
}

func TestImageGrouper_GroupImages_SingleImage(t *testing.T) {
	g := NewImageGrouper()
	session := &VisionSession{
		Images: []*PendingImage{
			{Filename: "test.jpg", MessageID: "msg1"},
		},
	}
	groups := g.GroupImages(session)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0]) != 1 {
		t.Errorf("expected 1 image in group, got %d", len(groups[0]))
	}
}

func TestImageGrouper_GroupImages_SameMessage(t *testing.T) {
	g := NewImageGrouper()
	session := &VisionSession{
		Images: []*PendingImage{
			{Filename: "IMG_001.JPG", MessageID: "msg1", DiscordTS: 1000.0},
			{Filename: "IMG_002.JPG", MessageID: "msg1", DiscordTS: 1000.0},
			{Filename: "IMG_003.JPG", MessageID: "msg2", DiscordTS: 2000.0},
		},
	}
	groups := g.GroupImages(session)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups (by message ID), got %d", len(groups))
	}

	// Find group with 2 images
	found := false
	for _, group := range groups {
		if len(group) == 2 {
			found = true
			for _, img := range group {
				if img.MessageID != "msg1" {
					t.Errorf("expected all images in 2-image group to have msg1, got %q", img.MessageID)
				}
			}
		}
	}
	if !found {
		t.Error("expected one group with 2 images")
	}
}

func TestImageGrouper_GroupImages_ExifProximity(t *testing.T) {
	g := NewImageGrouper()

	now := 1700000000.0
	session := &VisionSession{
		Images: []*PendingImage{
			{Filename: "IMG_001.JPG", ExifTS: now, DiscordTS: now},
			{Filename: "IMG_002.JPG", ExifTS: now + 30.0, DiscordTS: now + 30.0}, // 30s apart, same group
			{Filename: "IMG_003.JPG", ExifTS: now + 200.0, DiscordTS: now + 200.0}, // 200s apart, different group
		},
	}

	groups := g.GroupImages(session)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups (EXIF proximity), got %d", len(groups))
	}
}

func TestImageGrouper_GroupImages_DiscordFallback(t *testing.T) {
	g := NewImageGrouper()

	now := 1700000000.0
	session := &VisionSession{
		Images: []*PendingImage{
			{Filename: "IMG_001.JPG", ExifTS: 0, DiscordTS: now},             // no EXIF
			{Filename: "IMG_002.JPG", ExifTS: 0, DiscordTS: now + 60.0},      // 60s, within MessageGroupingWindow
			{Filename: "IMG_003.JPG", ExifTS: 0, DiscordTS: now + 500.0},     // 500s, separate group
		},
	}

	groups := g.GroupImages(session)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups (Discord timestamp fallback), got %d", len(groups))
	}
}

func TestImageGrouper_GroupImages_AllSeparate(t *testing.T) {
	g := NewImageGrouper()

	session := &VisionSession{
		Images: []*PendingImage{
			{Filename: "A.JPG", MessageID: "m1", ExifTS: 1000.0, DiscordTS: 1000.0},
			{Filename: "B.JPG", MessageID: "m2", ExifTS: 2000.0, DiscordTS: 2000.0},
			{Filename: "C.JPG", MessageID: "m3", ExifTS: 3000.0, DiscordTS: 3000.0},
		},
	}

	groups := g.GroupImages(session)
	if len(groups) != 3 {
		t.Fatalf("expected 3 separate groups, got %d", len(groups))
	}
}

func TestPendingImage_BestTimestamp_ExifPreferred(t *testing.T) {
	img := &PendingImage{ExifTS: 1000.0, DiscordTS: 2000.0}
	if img.BestTimestamp() != 1000.0 {
		t.Errorf("BestTimestamp() = %f, want 1000.0 (EXIF preferred)", img.BestTimestamp())
	}
}

func TestPendingImage_BestTimestamp_DiscordFallback(t *testing.T) {
	img := &PendingImage{ExifTS: 0, DiscordTS: 2000.0}
	if img.BestTimestamp() != 2000.0 {
		t.Errorf("BestTimestamp() = %f, want 2000.0 (Discord fallback)", img.BestTimestamp())
	}
}

func TestPendingImage_BestTimestamp_Both0(t *testing.T) {
	img := &PendingImage{}
	if img.BestTimestamp() != 0 {
		t.Errorf("BestTimestamp() = %f, want 0", img.BestTimestamp())
	}
}

func TestGroupingConstants(t *testing.T) {
	if ExifGroupingWindow != 60.0 {
		t.Errorf("ExifGroupingWindow = %f, want 60.0", ExifGroupingWindow)
	}
	if MessageGroupingWindow != 120.0 {
		t.Errorf("MessageGroupingWindow = %f, want 120.0", MessageGroupingWindow)
	}
}

func TestCategoryConstants(t *testing.T) {
	categories := []string{
		CategoryMotherboard, CategoryCPU, CategoryRAM, CategoryGPU,
		CategoryStorage, CategoryNIC, CategoryPSU, CategoryCase,
		CategoryCable, CategoryCooling, CategoryPeripheral, CategoryOther,
	}
	if len(categories) != 12 {
		t.Errorf("expected 12 category constants, got %d", len(categories))
	}

	// Verify no empty constants
	for _, cat := range categories {
		if cat == "" {
			t.Error("category constant should not be empty")
		}
	}
}
