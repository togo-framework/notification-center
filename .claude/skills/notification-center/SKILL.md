---
name: notification-center
description: Add an in-app notification inbox/bell to a togo app — store per-user notifications, list/filter, unread counts, mark read, delete (the database channel that complements the notifications plugin).
---

# togo notification-center

Use this skill to add an in-app notification inbox to a togo app.

## Send a notification
```go
nc, _ := notificationcenter.FromKernel(k)
nc.Notify(ctx, userID, notificationcenter.Notification{Type:"comment", Title:"New comment", Body:"...", ActionURL:"/posts/9"})
```

## Render a bell + inbox
- Badge count: `GET /api/notifications/unread-count` → `{count}`
- Inbox: `GET /api/notifications?unread=true` (newest first)
- Mark read: `POST /api/notifications/{id}/read` · all: `POST /api/notifications/read-all`
- Delete: `DELETE /api/notifications/{id}`
The acting user is the `X-User-Id` header (or an auth claim).

## Notes
- Complements `notifications` (external delivery: email/Slack/push) — this is the in-app store.
- Default store is bounded in-memory; implement the `Store` interface + `nc.WithStore(...)` for DB persistence.
- Wire `Notify` to your realtime broker to push live bell updates.
