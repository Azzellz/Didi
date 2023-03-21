package Delegator

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"time"
)

const (
	Normal = iota
	Cycle
	max
	unEffective
)

type Delegator interface {
	Cores         //核心功能
	Concurrency   //异步委托
	TimeOperation //时间操作
	CarryPattern  //执行模式
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
	Genshin()
	ShowMembers(pattern int)
	GetReturns() Returner
	back() *delegator
}

// Concurrency 异步接口
type Concurrency interface {
	Stop()
	Start()
	Wait()
	Over()
}

// TimeOperation 时间操作的接口
type TimeOperation interface {
	SetTime(t time.Duration, index int) Delegator
}

// CarryPattern 执行模式的接口
type CarryPattern interface {
	SetPattern(pattern int, args ...interface{}) Delegator
}

// Returner 返回值获取
type Returner interface {
	Get(fid int, pid int) (interface{}, error)
}

// 委托实体
type delegator struct {
	fs      []func() []reflect.Value //函数队列
	stop    chan int                 //无缓存管道,用来阻塞委托
	start   chan int                 //无缓存管道,用来恢复委托
	signal  chan int                 //委托完成的信号
	over    chan int                 //终止委托的信号
	returns chan returner            //存储返回值的管道
	names   []string                 //记录函数名
	sleeper []time.Duration          //目标函数需要睡眠的时间
	ptn     *pattern                 //执行模式
	typ     bool                     //判断是否为异步委托
}

type pattern struct {
	carryPattern int
	args         []interface{}
}

// 管理返回值的数据结构
type returner struct {
	vals map[int]map[int]interface{}
}

// New 根据参数生成不同运行模式的委托实体
func New(args ...interface{}) Delegator {
	if len(args) == 0 {
		return &delegator{fs: make([]func() []reflect.Value, 0), names: make([]string, 0), stop: make(chan int), signal: make(chan int), over: make(chan int), start: make(chan int), returns: make(chan returner, 1), typ: false, ptn: &pattern{
			carryPattern: 0, //默认是0
			args:         make([]interface{}, 0),
		}}
	} else {
		return &delegator{fs: make([]func() []reflect.Value, 0), names: make([]string, 0), stop: make(chan int), signal: make(chan int), over: make(chan int), start: make(chan int), returns: make(chan returner, 1), typ: true, ptn: &pattern{
			carryPattern: 0, //默认是0
			args:         make([]interface{}, 0),
		}}
	}
}

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
	d.sleeper = append(d.sleeper, 0)

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
		select {
		case <-d.signal:
		}
	}
}

func (d *delegator) Over() {
	if d.typ {
		d.over <- 1
	}
}

// 隐藏Run的细节
func (d *delegator) run() {
	//顺序执行
	//浅浅来个简单的异步执行
	returner := returner{vals: make(map[int]map[int]interface{})}
	//让委托执行的在一个协程里,方便委托中断
	for i, f := range d.fs {
		//select阻塞器,只有异步委托才会用上
		select {
		case <-d.stop:
			<-d.start
		default:
		}
		//睡眠的优先级要在阻塞之后
		time.Sleep(d.sleeper[i])

		returner.vals[i] = make(map[int]interface{})
		returnVals := f()
		for j, v2 := range returnVals {
			returner.vals[i][j] = v2.Interface()
		}
	}
	d.returns <- returner
}

// Run 执行委托,β型委托传参激活goroutine执行
func (d *delegator) Run(params ...interface{}) {
	if params == nil || !d.typ {
		//确定执行模式
		switch d.ptn.carryPattern {
		case Normal:
			d.run()
		case Cycle:
			//先确定ptn中的参数,
			//这里有个短路或,只要args长度为0,就不会再执行后面的语句了,就不会发生越界访问
			if len(d.ptn.args) == 0 || d.ptn.args[0] == -1 {
				for {
					select {
					case <-d.over:
						break
					default:
					}
					d.run()
					<-d.returns
				}
			} else {
				count := 0
				p := d.ptn.args[0]
				switch v := p.(type) {
				case string:
					count, _ = strconv.Atoi(v)
				case int:
					count = v
				default:
					//不是int,string类型的参数则只执行一次,表示循环无效
					count = 1
				}
				for i := 0; i < count; i++ {
					if i != 0 {
						//这里要做下清空管道操作,防止死锁
						<-d.returns
					}
					d.run()

				}
			}

		}

	} else if params != nil && d.typ {
		go func() {
			//确定执行模式
			switch d.ptn.carryPattern {
			case Normal:
				d.run()
			case Cycle:
				//先确定ptn中的参数,
				//这里有个短路或,只要args长度为0,就不会再执行后面的语句了,就不会发生越界访问
				if len(d.ptn.args) == 0 || d.ptn.args[0] == -1 {

					for {
						select {
						case <-d.over:
							break
						default:
						}
						d.run()
						//这里要做下清空管道操作,防止死锁
						<-d.returns
					}
				} else {
					count := 0
					p := d.ptn.args[0]
					switch v := p.(type) {
					case string:
						count, _ = strconv.Atoi(v)
					case int:
						count = v
					default:
						//不是int,string类型的参数则只执行一次,表示循环无效
						count = 1
					}
					for i := 0; i < count; i++ {
						if i != 0 {
							<-d.returns
						}
						d.run()
						//这里要做下清空管道操作,防止死锁
					}
				}
			}

			d.signal <- 1
		}()
	}

}

// GetReturns 获取返回值管理单元,自带同步阻塞,会等到委托执行完毕
func (d *delegator) GetReturns() Returner {
	//先判断是否执行完毕,没执行完毕会等待执行完毕
	tmp := <-d.returns
	return &tmp
}

func (d *delegator) back() *delegator {
	return d
}

// Join 连接另一个委托,叠加两个委托的函数队列
func (d *delegator) Join(d2 Delegator) Delegator {
	//要同步一下等待组的代办数和函数名
	d.fs = append(d.fs, d2.back().fs...)
	d.sleeper = append(d.sleeper, d2.back().sleeper...)
	d.names = append(d.names, d2.back().names...)

	return d
}

// SetTime 设置指定函数的睡眠时间,index参数为负数或超出委托成员长度则全部设置等长睡眠
func (d *delegator) SetTime(t time.Duration, index int) Delegator {
	//先初始化一下睡眠数组的切片
	if index < 0 || index > len(d.names)-1 {
		for i := range d.sleeper {
			d.sleeper[i] = t
		}
	} else {
		d.sleeper[index] = t
	}
	return d
}

func (d *delegator) SetPattern(pattern int, args ...interface{}) Delegator {
	if pattern < 0 || pattern > max {
		d.ptn.carryPattern = unEffective
	} else {
		d.ptn.carryPattern = pattern
		d.ptn.args = args
	}
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
