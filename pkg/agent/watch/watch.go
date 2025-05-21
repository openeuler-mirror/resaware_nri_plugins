package watch

import "k8s.io/apimachinery/pkg/watch"

type (
	EventType = watch.EventType
	Interface = watch.Interface
	Event     = watch.Event
)

const (
	Added    = watch.Added
	Modified = watch.Modified
	Error    = watch.Error
	Deleted  = watch.Deleted
)
