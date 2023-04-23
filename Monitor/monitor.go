package Monitor

import "time"

type Monitor interface {
	BackGround(f func())
	Stop()
	Start()
}

type monitor struct {
	stop  chan interface{}
	start chan interface{}
	flag  bool
	task  func()
	only  bool
}

func New() Monitor {
	return &monitor{stop: make(chan interface{}), start: make(chan interface{}), only: false}
}
func (w *monitor) BackGround(f func()) {

	if w.task != nil {
		w.only = true
	}

	if !w.only {
		w.task = f
		w.work()
	}

}

func (w *monitor) work() {
	go func() {
		for {
			if w.flag {
				select {
				case <-w.stop:
					<-w.start
					w.flag = false
				case <-time.After(1 * time.Second):
				}
			}
			w.task()
		}
	}()
}

func (w *monitor) Stop() {
	w.flag = true
	w.stop <- 0

}

func (w *monitor) Start() {
	w.flag = true
	w.start <- 1
}
