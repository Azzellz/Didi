package Pool

import (
	"fmt"
	"sync"
)

// TaskFunc 用来标记任务的函数
type TaskFunc func()

// Pool 线程池接口
type Pool interface {
	Len() int
	Assign(fs ...TaskFunc)
	Wait()
	Trigger()
	Now() int //返回当前空闲数
}

type worker struct {
	task     TaskFunc
	isAssign bool
}

func (w *worker) Do(f TaskFunc) {
	w.task = f
	w.task()
	w.task = nil
}

type pool struct {
	workers []worker       //任务队列
	cap     int            //线程池容量
	live    bool           //线程池状态
	wg      sync.WaitGroup //等待组
}

func New(cap int) Pool {

	if cap <= 0 {
		fmt.Println(" wrong cap ! please again ")
		return nil
	}

	p := &pool{
		workers: make([]worker, cap),
		cap:     cap,
		live:    true,
	}

	return p
}

func (p *pool) Assign(fs ...TaskFunc) {
	if !p.live {
		return
	}
	//遍历空闲worker队列,执行一次异步分配
	for i := range p.workers {

		w := &p.workers[i]

		if !w.isAssign {

			w.isAssign = true //标记为已分配

			p.wg.Add(len(fs))

			for _, f := range fs {
				go func() {

					defer func() {
						w.isAssign = false
						p.wg.Done()
					}()

					w.Do(f)

				}()
			}

			return //结束分配
		}
	}
}

// Wait 同步阻塞主线程,等待所有worker完成任务
func (p *pool) Wait() {
	p.wg.Wait()
}

// Trigger 若线程池活跃则关闭,若关闭则开启
func (p *pool) Trigger() {
	if p.live {
		p.live = false
	} else {
		p.live = true
	}
}

func (p *pool) Len() int {
	return len(p.workers)
}

func (p *pool) Now() int {
	count := 0
	for _, w := range p.workers {
		if w.isAssign {
			count++
		}
	}
	return count
}
