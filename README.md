<div align="center">
  <img src=".github/assets/togo-mark.svg" alt="togo" height="64" />
  <h1>togo-framework/notification-center</h1>
  <p>
    <a href="https://to-go.dev/marketplace"><img src="https://img.shields.io/badge/marketplace-to--go.dev-1FC7DC" alt="marketplace" /></a>
    <a href="https://pkg.go.dev/github.com/togo-framework/notification-center"><img src="https://pkg.go.dev/badge/github.com/togo-framework/notification-center.svg" alt="pkg.go.dev" /></a>
    <img src="https://img.shields.io/badge/license-MIT-blue" alt="MIT" />
  </p>
  <p><strong>In-app notification inbox for <a href="https://to-go.dev">togo</a> — the database channel + bell.</strong></p>
</div>

## Install

```bash
togo install togo-framework/notification-center
```

The togo answer to Laravel's **database notifications** / an in-app notification **bell**. It complements [`notifications`](https://to-go.dev/plugins/notifications) (which delivers to external providers like email/Slack/push) by storing a **per-user inbox** you can list, count, and mark read.

## Usage

```go
nc, _ := notificationcenter.FromKernel(k)

// Store a notification for a user.
nc.Notify(ctx, "user-1", notificationcenter.Notification{
    Type:      "comment",
    Title:     "New comment",
    Body:      "Sam replied to your post",
    ActionURL: "/posts/9",
    Data:      map[string]any{"post_id": 9},
})

unread := nc.UnreadCount("user-1")               // bell badge
items  := nc.List("user-1", notificationcenter.Filter{UnreadOnly: true})
nc.MarkRead(items[0].ID)
nc.MarkAllRead("user-1")
nc.Delete(items[0].ID)
```

## REST API

The acting user is resolved from the `X-User-Id` header (or an auth claim on the context).

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/notifications?unread=true` | List the inbox (newest first) |
| `GET` | `/api/notifications/unread-count` | `{ "count": N }` for the bell |
| `POST` | `/api/notifications/{id}/read` | Mark one read |
| `POST` | `/api/notifications/read-all` | Mark all read |
| `DELETE` | `/api/notifications/{id}` | Delete one |

## Configuration

No required env. Storage defaults to a bounded in-memory store; swap a DB-backed
implementation with `nc.WithStore(myStore)` (implements the `Store` interface).
If a realtime broker is configured, wire `Notify` to publish so the bell updates live.

---

<div align="center">
  <h3>Premium sponsors</h3>
  <p>
    <a href="https://id8media.com"><strong>ID8 Media</strong></a> &nbsp;·&nbsp;
    <a href="https://one-studio.co"><strong>One Studio</strong></a>
  </p>
  <p><sub>Support togo — <a href="https://github.com/sponsors/fadymondy">become a sponsor</a>.</sub></p>
</div>
