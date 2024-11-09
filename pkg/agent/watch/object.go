package watch

import (
	"context"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"
)

type CreateFn func(ctx context.Context, ns, name string) (watch.Interface, error)

const (
	reopenDelay = 5 * time.Second
)

type ObjectWatch struct {
	create    CreateFn
	ctx       context.Context
	namespace string
	name      string
	resultC   chan watch.Event
	wif       watch.Interface
	reopenC   <-chan time.Time
	failing   bool

	stopLock sync.Mutex
	stopC    chan struct{}
	doneC    chan struct{}
}

func Object(ctx context.Context, ns, name string, create CreateFn) (watch.Interface, error) {
	w := &ObjectWatch{
		create:    create,
		ctx:       ctx,
		namespace: ns,
		name:      name,
		resultC:   make(chan watch.Event, watch.DefaultChanSize),
		stopC:     make(chan struct{}),
		doneC:     make(chan struct{}),
	}

	if err := w.run(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *ObjectWatch) Stop() {
	w.stopLock.Lock()
	defer w.stopLock.Unlock()

	if w.stopC != nil {
		close(w.stopC)
		_ = <-w.doneC
		w.stopC = nil
	}
}

func (w *ObjectWatch) ResultChan() <-chan watch.Event {
	return w.resultC
}

func (w *ObjectWatch) eventChan() <-chan watch.Event {
	if w.wif == nil {
		return nil
	}
	return w.wif.ResultChan()
}

func (w *ObjectWatch) run() error {
	err := w.open()
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-w.stopC:
				w.Stop()
				close(w.resultC)
				close(w.doneC)
				return

			case e, ok := <-w.eventChan():
				if !ok {
					klog.Warningf("watch %s expired", w.watchname())
					w.reopen()
					continue
				}
				if e.Type == watch.Error {
					w.markFailing()
					w.stop()
				}
				select {
				case w.resultC <- e:
				default:
					w.markFailing()
					w.stop()
					w.scheduleReopen()
				}

			case <-w.reopenC:
				w.reopenC = nil
				w.reopen()
			}
		}
	}()

	return nil
}

func (w *ObjectWatch) open() error {
	if w.wif != nil {
		w.wif.Stop()
		w.wif = nil
	}

	wif, err := w.create(w.ctx, w.namespace, w.name)
	if err != nil {
		return err
	}

	w.wif = wif
	w.markRunning()
	return nil
}

func (w *ObjectWatch) reopen() error {
	if err := w.open(); err != nil {
		w.scheduleReopen()
	} else {
		klog.Infof("watch %s reopened", w.watchname())
		w.markRunning()
	}
	return nil
}

func (w *ObjectWatch) stop() {
	if w.wif != nil {
		w.wif.Stop()
		w.wif = nil
	}
}

func (w *ObjectWatch) scheduleReopen() {
	if w.resultC != nil {
		return
	}
	w.reopenC = time.After(reopenDelay)
}

func (w *ObjectWatch) markFailing() {
	if !w.failing {
		klog.Errorf("watch %s is now failing", w.watchname())
		w.failing = true
	}
}

func (w *ObjectWatch) markRunning() {
	if w.failing {
		klog.Errorf("watch %s is now running", w.watchname())
		w.failing = false
	}
}

func (w *ObjectWatch) watchname() string {
	if w.namespace != "" {
		return w.namespace + "/" + w.name
	}
	return w.name
}
