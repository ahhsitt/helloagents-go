package memory

import (
	"testing"
	"time"
)

func TestNewMemoryItem(t *testing.T) {
	content := "test content"
	memType := MemoryTypeWorking

	item := NewMemoryItem(content, memType)

	if item.Content != content {
		t.Errorf("expected content %q, got %q", content, item.Content)
	}
	if item.MemoryType != memType {
		t.Errorf("expected memory type %q, got %q", memType, item.MemoryType)
	}
	if item.ID == "" {
		t.Error("expected non-empty ID")
	}
	if item.Importance != 0.5 {
		t.Errorf("expected default importance 0.5, got %f", item.Importance)
	}
	if item.Metadata == nil {
		t.Error("expected non-nil metadata map")
	}
	if item.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestNewMemoryItemWithOptions(t *testing.T) {
	content := "test content"
	memType := MemoryTypeEpisodic
	userID := "user123"
	importance := float32(0.8)
	now := time.Now()
	metadata := map[string]interface{}{"key": "value"}

	item := NewMemoryItem(content, memType,
		WithUserID(userID),
		WithImportance(importance),
		WithTimestamp(now),
		WithMetadata(metadata),
	)

	if item.UserID != userID {
		t.Errorf("expected user ID %q, got %q", userID, item.UserID)
	}
	if item.Importance != importance {
		t.Errorf("expected importance %f, got %f", importance, item.Importance)
	}
	if !item.Timestamp.Equal(now) {
		t.Errorf("expected timestamp %v, got %v", now, item.Timestamp)
	}
	if item.Metadata["key"] != "value" {
		t.Errorf("expected metadata key=value, got %v", item.Metadata)
	}
}

func TestWithImportanceClamp(t *testing.T) {
	tests := []struct {
		name       string
		input      float32
		expected   float32
	}{
		{"negative", -0.5, 0},
		{"zero", 0, 0},
		{"normal", 0.5, 0.5},
		{"one", 1.0, 1.0},
		{"above one", 1.5, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := NewMemoryItem("test", MemoryTypeWorking, WithImportance(tt.input))
			if item.Importance != tt.expected {
				t.Errorf("expected importance %f, got %f", tt.expected, item.Importance)
			}
		})
	}
}

func TestWithID(t *testing.T) {
	customID := "custom-id-123"
	item := NewMemoryItem("test", MemoryTypeWorking, WithID(customID))

	if item.ID != customID {
		t.Errorf("expected ID %q, got %q", customID, item.ID)
	}
}

func TestWithMetadataKV(t *testing.T) {
	item := NewMemoryItem("test", MemoryTypeWorking,
		WithMetadataKV("key1", "value1"),
		WithMetadataKV("key2", 123),
	)

	if item.Metadata["key1"] != "value1" {
		t.Errorf("expected key1=value1, got %v", item.Metadata["key1"])
	}
	if item.Metadata["key2"] != 123 {
		t.Errorf("expected key2=123, got %v", item.Metadata["key2"])
	}
}

func TestMemoryItemValidate(t *testing.T) {
	tests := []struct {
		name      string
		item      *MemoryItem
		wantError bool
	}{
		{
			name:      "valid item",
			item:      NewMemoryItem("content", MemoryTypeWorking),
			wantError: false,
		},
		{
			name: "empty content",
			item: &MemoryItem{
				ID:         "id",
				Content:    "",
				MemoryType: MemoryTypeWorking,
				Importance: 0.5,
			},
			wantError: true,
		},
		{
			name: "empty memory type",
			item: &MemoryItem{
				ID:         "id",
				Content:    "content",
				MemoryType: "",
				Importance: 0.5,
			},
			wantError: true,
		},
		{
			name: "negative importance",
			item: &MemoryItem{
				ID:         "id",
				Content:    "content",
				MemoryType: MemoryTypeWorking,
				Importance: -0.1,
			},
			wantError: true,
		},
		{
			name: "importance above 1",
			item: &MemoryItem{
				ID:         "id",
				Content:    "content",
				MemoryType: MemoryTypeWorking,
				Importance: 1.1,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.item.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestMemoryItemClone(t *testing.T) {
	original := NewMemoryItem("content", MemoryTypeEpisodic,
		WithUserID("user1"),
		WithImportance(0.9),
		WithMetadataKV("key", "value"),
	)

	cloned := original.Clone()

	// Check values are equal
	if cloned.ID != original.ID {
		t.Errorf("expected ID %q, got %q", original.ID, cloned.ID)
	}
	if cloned.Content != original.Content {
		t.Errorf("expected content %q, got %q", original.Content, cloned.Content)
	}
	if cloned.MemoryType != original.MemoryType {
		t.Errorf("expected memory type %q, got %q", original.MemoryType, cloned.MemoryType)
	}
	if cloned.UserID != original.UserID {
		t.Errorf("expected user ID %q, got %q", original.UserID, cloned.UserID)
	}
	if cloned.Importance != original.Importance {
		t.Errorf("expected importance %f, got %f", original.Importance, cloned.Importance)
	}

	// Check metadata is a deep copy
	cloned.Metadata["key"] = "modified"
	if original.Metadata["key"] == "modified" {
		t.Error("expected metadata to be a deep copy")
	}
}

func TestGetMetadataString(t *testing.T) {
	item := NewMemoryItem("test", MemoryTypeWorking,
		WithMetadataKV("string", "value"),
		WithMetadataKV("int", 123),
	)

	if v := item.GetMetadataString("string"); v != "value" {
		t.Errorf("expected 'value', got %q", v)
	}
	if v := item.GetMetadataString("int"); v != "" {
		t.Errorf("expected empty string for non-string value, got %q", v)
	}
	if v := item.GetMetadataString("nonexistent"); v != "" {
		t.Errorf("expected empty string for nonexistent key, got %q", v)
	}
}

func TestGetMetadataFloat(t *testing.T) {
	item := NewMemoryItem("test", MemoryTypeWorking,
		WithMetadataKV("float64", float64(1.5)),
		WithMetadataKV("float32", float32(2.5)),
		WithMetadataKV("int", 3),
		WithMetadataKV("int64", int64(4)),
		WithMetadataKV("string", "not a number"),
	)

	tests := []struct {
		key      string
		expected float64
	}{
		{"float64", 1.5},
		{"float32", 2.5},
		{"int", 3.0},
		{"int64", 4.0},
		{"string", 0},
		{"nonexistent", 0},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if v := item.GetMetadataFloat(tt.key); v != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, v)
			}
		})
	}
}

func TestAgeHours(t *testing.T) {
	now := time.Now()
	twoHoursAgo := now.Add(-2 * time.Hour)

	item := NewMemoryItem("test", MemoryTypeWorking, WithTimestamp(twoHoursAgo))

	ageHours := item.AgeHours()
	if ageHours < 1.9 || ageHours > 2.1 {
		t.Errorf("expected age around 2 hours, got %f", ageHours)
	}
}

func TestAgeDays(t *testing.T) {
	now := time.Now()
	twoDaysAgo := now.Add(-48 * time.Hour)

	item := NewMemoryItem("test", MemoryTypeWorking, WithTimestamp(twoDaysAgo))

	ageDays := item.AgeDays()
	if ageDays < 1.9 || ageDays > 2.1 {
		t.Errorf("expected age around 2 days, got %f", ageDays)
	}
}

func TestMemoryTypes(t *testing.T) {
	if MemoryTypeWorking != "working" {
		t.Errorf("expected MemoryTypeWorking = 'working', got %q", MemoryTypeWorking)
	}
	if MemoryTypeEpisodic != "episodic" {
		t.Errorf("expected MemoryTypeEpisodic = 'episodic', got %q", MemoryTypeEpisodic)
	}
	if MemoryTypeSemantic != "semantic" {
		t.Errorf("expected MemoryTypeSemantic = 'semantic', got %q", MemoryTypeSemantic)
	}
}
