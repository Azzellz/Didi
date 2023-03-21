package Delegator

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"
)

type Delegator interface {
	Cores //核心功能
	Concurrency
	Genshin()
	ShowMembers(pattern int)
	GetReturns() Returner
	back() *delegator
}

//// Delegatorβ 支持并发的委托接口
//type Delegatorβ interface {
//	Concurrency //并发功能
//	Delegator      //原委托接口的功能
//}

// Cores 委托核心功能接口
type Cores interface {
	Load(f interface{}, args ...interface{}) Delegator
	Join(delegator2 Delegator) Delegator
	Run(params ...interface{})
}

// Concurrency 并发接口
type Concurrency interface {
	Stop()
	Start()
	Wait()
}

// Returner 返回值获取
type Returner interface {
	Get(fid int, pid int) (interface{}, error)
}

// 委托实体
type delegator struct {
	fs      []func() []reflect.Value //函数队列
	wg      sync.WaitGroup           //等待组
	stop    chan int                 //无缓存管道,用来阻塞委托
	start   chan int                 //无缓存管道,用来恢复委托
	over    chan int                 //用来当作结束的信号
	returns chan returner            //存储返回值的管道
	names   []string                 //记录函数名
	typ     bool                     //判断是否为β类型
}

// 管理返回值的数据结构
type returner struct {
	vals map[int]map[int]interface{}
}

// New 根据参数生成不同运行模式的委托实体
func New(args ...interface{}) Delegator {
	if len(args) == 0 {
		return &delegator{fs: make([]func() []reflect.Value, 0), names: make([]string, 0), stop: make(chan int), start: make(chan int), returns: make(chan returner, 1), over: make(chan int, 1), typ: false}
	} else {
		return &delegator{fs: make([]func() []reflect.Value, 0), names: make([]string, 0), stop: make(chan int), start: make(chan int), returns: make(chan returner, 1), over: make(chan int, 1), typ: true}
	}
}

// New_ 生成运行在goroutine上的委托
//func New_() Delegatorβ {
//	return &delegator{fs: make([]func() []reflect.Value, 0), names: make([]string, 0), stop: make(chan int), start: make(chan int), returns: make(chan returner, 1), over: make(chan int, 1), typ: true}
//}

// Load 为委托装载函数,f参数为目标函数,args为可选参数,按顺序识别,多于目标函数入参的参数无效.
func (d *delegator) Load(f interface{}, args ...interface{}) Delegator {
	//先获取函数的类型反射对象
	t := reflect.TypeOf(f)
	v := reflect.ValueOf(f)
	fName := runtime.FuncForPC(v.Pointer()).Name()
	//获取函数类型反射对象的入参数
	inParams := make([]reflect.Value, t.NumIn())

	for i := 0; i < t.NumIn(); i++ {
		inParams[i] = reflect.ValueOf(args[i])
	}

	//闭包
	d.fs = append(d.fs, func() []reflect.Value { return v.Call(inParams) })
	//为等待组添加等待数
	d.wg.Add(1)

	tmp := v.String()[1:]
	tmp = tmp[:len(tmp)-6]
	formatName := fmt.Sprintf("函数名:%s,签名:%s\n", fName, tmp)
	d.names = append(d.names, formatName)
	return d
}

// ShowMembers 分行打印当前委托装载的函数名和函数签名
func (d *delegator) ShowMembers(pattern int) {
	if pattern <= -1 || len(d.names) < pattern {
		for i, v := range d.names {
			fmt.Println(fmt.Sprintf("第%d位:%s", i+1, v))
		}
	} else {
		for i := 0; i < pattern; i++ {
			fmt.Println(fmt.Sprintf("第%d位:%s", i+1, d.names[i]))
		}
	}
}

func (d *delegator) Genshin() {
	fmt.Println("Genshin impact")
}

// Stop 暂停运行在goroutine上的委托,不会阻塞主协程
func (d *delegator) Stop() {
	if d.typ {
		d.stop <- 1
	}
}

// Start 启动运行在goroutine上的委托,不会阻塞主协程
func (d *delegator) Start() {
	if d.typ {
		d.start <- 1
	}
}

// Wait 等待委托执行完毕,会阻塞主协程
func (d *delegator) Wait() {
	if d.typ {
		d.wg.Wait()
	}
}

// Run 执行委托,β型委托传参激活goroutine执行
func (d *delegator) Run(params ...interface{}) {
	if params == nil || !d.typ {

		//顺序执行
		//浅浅来个简单的异步执行
		returner := returner{vals: make(map[int]map[int]interface{})}
		//让委托执行的在一个协程里,方便委托中断
		for i, f := range d.fs {

			returner.vals[i] = make(map[int]interface{})
			returnVals := f()
			for j, v2 := range returnVals {
				returner.vals[i][j] = v2.Interface()
			}

			d.wg.Done()
		}

		//over要放在后面
		d.returns <- returner
		d.over <- 1

	} else if params != nil && d.typ {
		go func() {
			//顺序执行
			//浅浅来个简单的异步执行
			returner := returner{vals: make(map[int]map[int]interface{})}
			//让委托执行的在一个协程里,方便委托中断
			for i, f := range d.fs {
				//select阻塞器
				select {
				case <-d.stop:
					<-d.start
				default:
				}

				returner.vals[i] = make(map[int]interface{})
				returnVals := f()
				for j, v2 := range returnVals {
					returner.vals[i][j] = v2.Interface()
				}

				d.wg.Done()
			}

			//over要放在后面
			d.returns <- returner
			d.over <- 1
		}()
	}

}

// GetReturns 获取返回值管理单元,自带同步阻塞,会等到委托执行完毕
func (d *delegator) GetReturns() Returner {
	//先判断是否执行完毕,没执行完毕会等待执行完毕
	<-d.over
	tmp := <-d.returns
	return &tmp
}

func (d *delegator) back() *delegator {
	return d
}

// Join 连接另一个委托,叠加两个委托的函数队列
func (d *delegator) Join(d2 Delegator) Delegator {
	d.fs = append(d.fs, d2.back().fs...)
	//要同步一下等待组的代办数和函数名
	d.wg.Add(len(d2.back().fs))
	d.names = append(d.names, d2.back().names...)

	return d
}

// Get 获取返回值的接口值,需要自行类型断言,fid为函数索引,pid为对应函数的返回值索引
func (r *returner) Get(fid int, pid int) (interface{}, error) {

	if f, ok := r.vals[fid]; ok {
		if p, ok := f[pid]; ok {
			return p, nil
		} else {
			return nil, fmt.Errorf("can't find target function,fid is wrong")
		}
	} else {
		return nil, fmt.Errorf("can't find target function,fid is wrong")
	}

}
