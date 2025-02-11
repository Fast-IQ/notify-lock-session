package notify_lock_session

import "time"

type NotifyLock struct {
}

type Lock struct {
	Lock  bool
	Clock time.Time
}
