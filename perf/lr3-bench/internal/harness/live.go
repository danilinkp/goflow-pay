package harness

import (
	"bufio"
	"io"
	"math/bits"
	"strconv"
	"sync/atomic"
	"time"
)

const (
	liveSubBits = 5
	liveSub     = 1 << liveSubBits
	liveBuckets = 64 * liveSub
)

var phaseNames = [...]string{"warmup", "measure", "teardown"}

type Live struct {
	out      *bufio.Writer
	interval time.Duration
	ops      []string
	counts   [3][][liveBuckets]atomic.Uint64
	prev     [3][][liveBuckets]uint64
}

func NewLive(out io.Writer, interval time.Duration, ops []string) *Live {
	l := &Live{out: bufio.NewWriter(out), interval: interval, ops: ops}
	for p := range l.counts {
		l.counts[p] = make([][liveBuckets]atomic.Uint64, len(ops))
		l.prev[p] = make([][liveBuckets]uint64, len(ops))
	}
	return l
}

func liveBucket(v int64) int {
	if v < liveSub {
		return int(max(v, 0))
	}
	u := uint64(v)
	shift := bits.Len64(u) - liveSubBits - 1
	return (shift+1)*liveSub + int(u>>shift) - liveSub
}

func liveBounds(b int) (int64, int64) {
	if b < liveSub {
		return int64(b), int64(b) + 1
	}
	shift := b/liveSub - 1
	top := int64(b%liveSub + liveSub)
	return top << shift, (top + 1) << shift
}

func (l *Live) record(op uint8, phase Phase, latencyNs int64) {
	l.counts[phase][op][liveBucket(latencyNs)].Add(1)
}

func (l *Live) flush() {
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	var buf []byte
	for p := range l.counts {
		for op := range l.counts[p] {
			for b := range l.counts[p][op] {
				cur := l.counts[p][op][b].Load()
				n := cur - l.prev[p][op][b]
				if n == 0 {
					continue
				}
				l.prev[p][op][b] = cur
				lo, hi := liveBounds(b)
				buf = append(buf[:0], ts...)
				buf = append(buf, ',')
				buf = append(buf, phaseNames[p]...)
				buf = append(buf, ',')
				buf = append(buf, l.ops[op]...)
				buf = append(buf, ',')
				buf = strconv.AppendInt(buf, int64(b), 10)
				buf = append(buf, ',')
				buf = strconv.AppendInt(buf, lo, 10)
				buf = append(buf, ',')
				buf = strconv.AppendInt(buf, hi, 10)
				buf = append(buf, ',')
				buf = strconv.AppendUint(buf, n, 10)
				buf = append(buf, '\n')
				l.out.Write(buf)
			}
		}
	}
	l.out.Flush()
}

func (l *Live) stream(stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	t := time.NewTicker(l.interval)
	defer t.Stop()
	for {
		select {
		case <-stop:
			l.flush()
			return
		case <-t.C:
			l.flush()
		}
	}
}
