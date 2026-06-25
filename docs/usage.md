# notification-center — usage

## Store a notification
```go
nc, _ := notificationcenter.FromKernel(k)
nc.Notify(ctx, userID, notificationcenter.Notification{
    Type: "comment", Title: "New comment", Body: "...", ActionURL: "/posts/9",
})
```

## Read the inbox
```go
nc.UnreadCount(userID)                                   // bell badge
nc.List(userID, notificationcenter.Filter{UnreadOnly:true, Limit:20, Offset:0})
nc.MarkRead(id); nc.MarkAllRead(userID); nc.Delete(id)
```

## REST
The acting user comes from the `X-User-Id` header (or an auth claim).

| Method | Path |
|---|---|
| GET | `/api/notifications?unread=true` |
| GET | `/api/notifications/unread-count` |
| POST | `/api/notifications/{id}/read` |
| POST | `/api/notifications/read-all` |
| DELETE | `/api/notifications/{id}` |

## Custom store
Implement the `Store` interface (Add/ByUser/Get/Delete) for DB persistence and
install it with `nc.WithStore(myStore)`. The default is a bounded in-memory store.
