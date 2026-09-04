package domain

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"simple name", "iPhone 15 Pro", "iphone-15-pro"},
		{"already lowercase", "wireless mouse", "wireless-mouse"},
		{"punctuation collapses to dashes", "Sony WH-1000XM5 Headphones!", "sony-wh-1000xm5-headphones"},
		{"leading and trailing spaces trimmed", "  Denim Jacket  ", "denim-jacket"},
		{"repeated separators collapse", "A---B   C", "a-b-c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := slugify(tt.in); got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSlugify_EmptyInputGeneratesUUID(t *testing.T) {
	got := slugify("   ")
	if len(got) != 36 {
		t.Errorf("slugify(empty) = %q, want a UUID-shaped fallback", got)
	}
}

func TestProduct_BeforeCreate(t *testing.T) {
	p := &Product{Name: "Wireless Gaming Mouse"}
	if err := p.BeforeCreate(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID.String() == "00000000-0000-0000-0000-000000000000" {
		t.Error("expected ID to be generated")
	}
	if p.Slug != "wireless-gaming-mouse" {
		t.Errorf("Slug = %q, want %q", p.Slug, "wireless-gaming-mouse")
	}
}

func TestProduct_BeforeCreate_PreservesExplicitSlug(t *testing.T) {
	p := &Product{Name: "Wireless Gaming Mouse", Slug: "custom-slug"}
	if err := p.BeforeCreate(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Slug != "custom-slug" {
		t.Errorf("Slug = %q, want explicit slug to be preserved", p.Slug)
	}
}

func TestCategory_BeforeCreate(t *testing.T) {
	c := &Category{Name: "Home & Garden"}
	if err := c.BeforeCreate(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Slug != "home-garden" {
		t.Errorf("Slug = %q, want %q", c.Slug, "home-garden")
	}
}
