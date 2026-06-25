package notificationcenter

import (
	"context"
	"testing"
)

func newTestService() *Service { return (&Service{}).WithStore(newMemStore()) }

func TestNotifyAndList(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	s.Notify(ctx, "u1", Notification{Title: "a"})
	s.Notify(ctx, "u1", Notification{Title: "b"})
	s.Notify(ctx, "u2", Notification{Title: "c"})

	if got := s.List("u1", Filter{}); len(got) != 2 {
		t.Fatalf("u1 should have 2 notifications, got %d", len(got))
	}
	if got := s.List("u2", Filter{}); len(got) != 1 || got[0].Title != "c" {
		t.Fatalf("u2 list = %+v", got)
	}
	// Per-user isolation: u2 must not see u1's items.
	for _, n := range s.List("u2", Filter{}) {
		if n.UserID != "u2" {
			t.Fatalf("isolation leak: %+v", n)
		}
	}
}

func TestUnreadCountAndMarkRead(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	a := s.Notify(ctx, "u1", Notification{Title: "a"})
	s.Notify(ctx, "u1", Notification{Title: "b"})

	if c := s.UnreadCount("u1"); c != 2 {
		t.Fatalf("unread = %d, want 2", c)
	}
	if !s.MarkRead(a.ID) {
		t.Fatal("MarkRead returned false for an existing id")
	}
	if c := s.UnreadCount("u1"); c != 1 {
		t.Fatalf("unread after one read = %d, want 1", c)
	}
	if s.MarkRead("does-not-exist") {
		t.Fatal("MarkRead should be false for an unknown id")
	}
	// Read item carries a ReadAt timestamp.
	got, _ := s.store.Get(a.ID)
	if !got.Read || got.ReadAt == nil {
		t.Fatalf("read item missing Read/ReadAt: %+v", got)
	}
}

func TestUnreadFilterAndMarkAll(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	s.Notify(ctx, "u1", Notification{Title: "a"})
	s.Notify(ctx, "u1", Notification{Title: "b"})
	s.Notify(ctx, "u1", Notification{Title: "c"})

	if got := s.List("u1", Filter{UnreadOnly: true}); len(got) != 3 {
		t.Fatalf("unread filter = %d, want 3", len(got))
	}
	if changed := s.MarkAllRead("u1"); changed != 3 {
		t.Fatalf("MarkAllRead changed %d, want 3", changed)
	}
	if got := s.List("u1", Filter{UnreadOnly: true}); len(got) != 0 {
		t.Fatalf("after mark-all, unread = %d, want 0", len(got))
	}
	if c := s.UnreadCount("u1"); c != 0 {
		t.Fatalf("unread after mark-all = %d", c)
	}
}

func TestDelete(t *testing.T) {
	s := newTestService()
	n := s.Notify(context.Background(), "u1", Notification{Title: "x"})
	if !s.Delete(n.ID) {
		t.Fatal("Delete returned false")
	}
	if len(s.List("u1", Filter{})) != 0 {
		t.Fatal("notification not removed")
	}
	if s.Delete(n.ID) {
		t.Fatal("second Delete should be false")
	}
}

func TestListNewestFirstAndPaging(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	for _, ttl := range []string{"1", "2", "3", "4", "5"} {
		s.Notify(ctx, "u1", Notification{Title: ttl})
	}
	page := s.List("u1", Filter{Limit: 2})
	if len(page) != 2 {
		t.Fatalf("limit 2 returned %d", len(page))
	}
	// Newest first: title "5" then "4".
	if page[0].Title != "5" || page[1].Title != "4" {
		t.Fatalf("ordering wrong: %s,%s", page[0].Title, page[1].Title)
	}
	if off := s.List("u1", Filter{Limit: 2, Offset: 4}); len(off) != 1 {
		t.Fatalf("offset page = %d, want 1", len(off))
	}
}
