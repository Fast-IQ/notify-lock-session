package notify_lock_session

import "time"

type Lock struct {
	Lock  bool
	Clock time.Time
}
