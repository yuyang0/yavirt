package boar

import (
	"context"
	"fmt"
	"sync"

	"github.com/panjf2000/ants/v2"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/idgen"
	"github.com/projecteru2/yavirt/pkg/log"
)

const (
	destroyOp         op = "destroy"
	shutOp            op = "shutdown"
	bootOp            op = "boot"
	createOp          op = "create"
	resizeOp          op = "resize"
	miscOp            op = "misc"
	createSnapshotOp  op = "create-snapshot"
	commitSnapshotOp  op = "commit-snapshot"
	restoreSnapshotOp op = "restore-snapshot"
)

type op string

func (op op) String() string {
	return string(op)
}

type task struct {
	sync.Mutex

	id      string
	guestID string
	op      op
	do      func(context.Context) (any, error)
	res     any
	done    chan struct{}
	once    sync.Once
	err     error
}

func newTask(id string, op op, fn doFunc) *task {
	return &task{
		id:      idgen.Next(),
		guestID: id,
		op:      op,
		do:      fn,
		done:    make(chan struct{}),
	}
}

// String .
func (t *task) String() string {
	return fmt.Sprintf("%s <%s, %s>", t.id, t.op, t.guestID)
}

func (t *task) abort() {
	t.finish()
	t.err = errors.Trace(errors.ErrSerializedTaskAborted)
}

// terminate forcibly terminates a task.
func (t *task) terminate() { //nolint
	// TODO
}

func (t *task) run(ctx context.Context) error {
	defer t.finish()

	var res any
	var err error

	select {
	case <-ctx.Done():
		err = ctx.Err()
	default:
		res, err = t.do(ctx)
	}
	t.setResult(res, err)
	return err
}

func (t *task) finish() {
	t.once.Do(func() {
		close(t.done)
	})
}

func (t *task) result() (any, error) {
	t.Lock()
	defer t.Unlock()
	return t.res, t.err
}

func (t *task) setResult(res any, err error) {
	t.Lock()
	defer t.Unlock()

	t.res, t.err = res, err
}

type taskPool struct {
	lck  sync.Mutex
	mgr  map[string]*task
	pool *ants.Pool
}

func newTaskPool(max int) (*taskPool, error) {
	p, err := ants.NewPool(max, ants.WithNonblocking(true))
	if err != nil {
		return nil, err
	}
	return &taskPool{
		pool: p,
		mgr:  make(map[string]*task),
	}, nil
}

func (p *taskPool) SubmitTask(ctx context.Context, t *task) (err error) {
	p.lck.Lock()
	defer p.lck.Unlock()
	p.mgr[t.id] = t

	err = p.pool.Submit(func() {
		if err := t.run(ctx); err != nil {
			log.ErrorStack(err)
			metrics.IncrError()

		}
	})
	if err != nil {
		delete(p.mgr, t.id)
	} else {
		metrics.Incr(metrics.MetricSvcTaskTotal, nil) //nolint:errcheck
		metrics.Incr(metrics.MetricSvcTasks, nil)     //nolint:errcheck
	}
	return
}

func (p *taskPool) doneTask(t *task) {
	p.lck.Lock()
	defer p.lck.Unlock()
	delete(p.mgr, t.id)
	metrics.Decr(metrics.MetricSvcTasks, nil) //nolint:errcheck
}

func (p *taskPool) Release() {
	p.pool.Release()
}
