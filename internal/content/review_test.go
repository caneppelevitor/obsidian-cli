package content

import (
	"strings"
	"testing"
)

func TestParseReviewItems(t *testing.T) {
	input := `# Review Queue

## Pending

- [ ] [[Systems Over Goals]]
- [ ] [[Breaking Patterns]]
- [x] [[Already Reviewed]]

## Approved

- [x] [[Old Approved]]
`

	items := ParseReviewItems(input)
	if len(items) != 2 {
		t.Fatalf("expected 2 pending items, got %d", len(items))
	}
	if items[0].Name != "Systems Over Goals" {
		t.Errorf("items[0].Name = %q, want %q", items[0].Name, "Systems Over Goals")
	}
	if items[1].Name != "Breaking Patterns" {
		t.Errorf("items[1].Name = %q, want %q", items[1].Name, "Breaking Patterns")
	}
}

func TestParseReviewItemsNoPending(t *testing.T) {
	input := "# Review Queue\n\n## Approved\n\n- [x] [[Done]]"
	items := ParseReviewItems(input)
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestParseReviewItemsEmpty(t *testing.T) {
	items := ParseReviewItems("")
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestApproveReviewItem(t *testing.T) {
	input := `## Pending

- [ ] [[Draft A]]
- [ ] [[Draft B]]
`

	result := ApproveReviewItem(input, "Draft A")

	if strings.Contains(result, "- [ ] [[Draft A]]") {
		t.Error("Draft A should be removed from Pending")
	}
	if !strings.Contains(result, "## Approved") {
		t.Error("Approved section should be created")
	}
	if !strings.Contains(result, "- [x] [[Draft A]]") {
		t.Error("Draft A should appear checked in Approved")
	}
	if !strings.Contains(result, "- [ ] [[Draft B]]") {
		t.Error("Draft B should remain in Pending")
	}
}

func TestDiscardReviewItem(t *testing.T) {
	input := `## Pending

- [ ] [[Draft A]]

## Approved

- [x] [[Old]]
`

	result := DiscardReviewItem(input, "Draft A")

	if strings.Contains(result, "- [ ] [[Draft A]]") {
		t.Error("Draft A should be removed from Pending")
	}
	if !strings.Contains(result, "## Discarded") {
		t.Error("Discarded section should be created")
	}
	if !strings.Contains(result, "- [x] [[Draft A]]") {
		t.Error("Draft A should appear checked in Discarded")
	}
}

func TestApproveNonexistentItem(t *testing.T) {
	input := "## Pending\n\n- [ ] [[Draft A]]\n"
	result := ApproveReviewItem(input, "Nonexistent")
	if result != input {
		t.Error("content should be unchanged when item not found")
	}
}
