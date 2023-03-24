package Delegator

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"time"
)

const (
	Normal  = iota //标准委托
	Cycle          //循环执行
	TimeOut        //延时执行
	Tick           //间隔执行
	max            //占位
	unEffective
)

type Delegator interface {
	cores         //核心功能
	concurrency   //异步委托
	timeOperation //时间操作
	carryPattern  //执行模式
}

// cores 委托核心功能接口
type cores interface {
	Load(f interface{}, args ...interface{}) Delegator
	Quick(fs ...interface{}) Delegator
	Same(f interface{}, num int, args ...interface{}) Delegator
	Join(delegator2 Delegator) Delegator
	Run(params ...interface{}) error
	Genshin()
	ShowMembers(pattern int)
	GetReturns() (Returner, error)
	back() *delegator
}

// concurrency 异步接口
type concurrency interface {
	Stop()
	Start()
	Wait()
	Over()
	Sleep(duration time.Duration)
}

// timeOperation 时间操作的接口
type timeOperation interface {
	SetTime(t time.Duration, index int) Delegator
}

// carryPattern 执行模式的接口
type carryPattern interface {
	SetPattern(pattern int, args ...interface{}) Delegator
}

// Returner 返回值获取
type Returner interface {
	Get(fid int, pid int) (interface{}, error)
	BackError() error
}

// 委托实体
type delegator struct {
	fs      []func() []reflect.Value //函数队列
	cs      *chans                   //管道集合
	names   []string                 //记录函数名
	sleeper []time.Duration          //目标函数需要睡眠的时间
	ptn     *pattern                 //执行模式
	err     error                    //记录错误,在委托run的时候返回
	typ     bool                     //判断是否为异步委托
	goRun   bool                     //判断是否激活了异步执行
}

type chans struct {
	stop    chan int      //无缓存管道,用来阻塞委托
	start   chan int      //无缓存管道,用来恢复委托
	signal  chan int      //委托完成的信号
	over    chan int      //终止委托的信号
	returns chan returner //存储返回值的管道
}

type pattern struct {
	carryPattern int
	args         []interface{}
}

// 管理返回值的数据结构
type returner struct {
	vals map[int]map[int]interface{}
	err  chan error //异步获取错误
}

// New 根据参数生成不同运行模式的委托实体
func New(args ...interface{}) Delegator {
	if len(args) == 0 {
		return &delegator{
			fs:    make([]func() []reflect.Value, 0),
			names: make([]string, 0),
			cs: &chans{
				stop:    make(chan int),
				start:   make(chan int),
				signal:  make(chan int),
				over:    make(chan int),
				returns: make(chan returner, 1),
			},
			typ:   false,
			goRun: false,
			ptn: &pattern{
				carryPattern: 0, //默认是0
				args:         make([]interface{}, 0),
			},
		}
	} else {
		return &delegator{
			fs:    make([]func() []reflect.Value, 0),
			names: make([]string, 0),
			cs: &chans{
				stop:    make(chan int),
				start:   make(chan int),
				signal:  make(chan int),
				over:    make(chan int),
				returns: make(chan returner, 1),
			},
			typ:   true,
			goRun: false,
			ptn: &pattern{
				carryPattern: 0, //默认是0
				args:         make([]interface{}, 0),
			},
		}
	}
}

// 委托的装载内核,妈的,要适配Same方法所以抽取出来了
func (d *delegator) load(f interface{}, args interface{}) Delegator {
	ars, ok := args.([]interface{})
	if ok {
		//先获取函数的类型反射对象
		t := reflect.TypeOf(f)
		v := reflect.ValueOf(f)
		fName := runtime.FuncForPC(v.Pointer()).Name()
		//获取函数类型反射对象的入参数
		inParams := make([]reflect.Value, t.NumIn())

		for i := 0; i < t.NumIn(); i++ {
			if i > len(ars)-1 {
				//不好好传参,就只能在这里给你们进行默认值处理,妈的,只处理了一些常见的类型
				switch t.In(i).Kind() {
				case reflect.Int:
					inParams[i] = reflect.ValueOf(0)
				case reflect.Float64, reflect.Float32:
					inParams[i] = reflect.ValueOf(0.0)
				case reflect.String:
					inParams[i] = reflect.ValueOf("")
				case reflect.Bool:
					inParams[i] = reflect.ValueOf(false)
				default:
					d.err = fmt.Errorf("error ! uncorrect in params , please check")
				}
			} else {
				inParams[i] = reflect.ValueOf(ars[i])
			}
		}

		//闭包
		d.fs = append(d.fs, func() []reflect.Value { return v.Call(inParams) })
		d.sleeper = append(d.sleeper, 0)

		tmp := v.String()[1:]
		tmp = tmp[:len(tmp)-6]
		formatName := fmt.Sprintf("函数名:%s,签名:%s\n", fName, tmp)
		d.names = append(d.names, formatName)
	}
	return d
}

// Load 为委托装载函数,f参数为目标函数,args为可选参数,按顺序识别,多于目标函数入参的参数无效.
func (d *delegator) Load(f interface{}, args ...interface{}) Delegator {
	if d.err != nil {
		return d
	}
	d.load(f, args)
	return d
}

// Quick 快速装填多个无入参的函数
func (d *delegator) Quick(fs ...interface{}) Delegator {
	if d.err != nil {
		return d
	}
	for _, f := range fs {
		d.Load(f)
	}
	return d
}

// Same 装填多个相同的函数
func (d *delegator) Same(f interface{}, num int, args ...interface{}) Delegator {
	if d.err != nil {
		return d
	}
	for i := 0; i < num; i++ {
		d.load(f, args)
	}
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
	if d.err != nil {
		return
	}
	if d.typ && d.goRun {
		d.cs.stop <- 1
	}
}

// Start 启动运行在goroutine上的委托,不会阻塞主协程
func (d *delegator) Start() {
	if d.err != nil {
		return
	}
	if d.typ && d.goRun {
		d.cs.start <- 1
	}
}

// Wait 等待委托执行完毕,会阻塞主协程
func (d *delegator) Wait() {
	if d.err != nil {
		return
	}
	if d.typ && d.goRun {
		select {
		case <-d.cs.signal:
		}
	}
}

// Over 终止委托
func (d *delegator) Over() {
	if d.err != nil {
		return
	}
	if d.typ && d.goRun {
		//这里要加select,防止委托过早执行完毕导致主线程写入阻塞
		select {
		case d.cs.over <- 1:
		default:
		}
	}
}

// Sleep 对stop和start的简单封装,因为委托走并发,所以不能保证立刻睡眠
func (d *delegator) Sleep(duration time.Duration) {
	if d.err != nil {
		return
	}
	if d.typ && d.goRun {
		go func() {
			d.Stop()
			time.Sleep(duration)
			d.Start()
		}()
	}
}

// 隐藏Run的细节
func (d *delegator) run() {

	//顺序执行
	returner := returner{vals: make(map[int]map[int]interface{}), err: make(chan error, 1)}
	//让委托执行的在一个协程里,方便委托中断
	for i, f := range d.fs {
		//select阻塞器,只有异步委托才会用上
		select {
		case <-d.cs.stop:
			<-d.cs.start
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
	d.cs.returns <- returner
}

// Run 执行委托,β型委托传参激活goroutine执行
func (d *delegator) Run(params ...interface{}) error {
	if d.err != nil {
		return d.err
	}
	if params == nil || !d.typ {
		//确定执行模式
		switch d.ptn.carryPattern {
		case Normal:
			select {
			case <-d.cs.over:
				return nil
			default:
			}
			d.run()
		case Cycle:
			//先确定ptn中的参数,
			//不传参默认无限循环
			if len(d.ptn.args) == 0 {
				for {
					select {
					case <-d.cs.over:
						return nil
					default:
					}
					d.run()
					<-d.cs.returns
				}
			} else {
				count := 0
				p := d.ptn.args[0]
				switch v := p.(type) {
				case int:
					count = v
				default:
					//不是int类型的参数则只执行一次,表示循环无效
					count = 1
				}
				for i := 0; i < count; i++ {
					select {
					case <-d.cs.over:
						return nil
					default:
					}
					if i != 0 {
						//这里要做下清空管道操作,防止死锁
						<-d.cs.returns
					}
					d.run()

				}
			}
		case TimeOut:
			select {
			case <-d.cs.over:
				return nil
			default:
			}
			//没有传参或者无效时间参数则模式无效,正常执行
			if len(d.ptn.args) == 0 {
				d.run()
				d.err = fmt.Errorf("error time param ! you need to put a time.Duration type in second empty")
				return d.err
			}
			t, ok := d.ptn.args[0].(time.Duration)
			if ok {
				time.Sleep(t)
			} else {
				d.err = fmt.Errorf("error time param ! you need to put a time.Duration type in second empty")
			}
			d.run()

		case Tick: //间隔循环执行,第一个参数是间隔时间,第二个参数是次数
			select {
			case <-d.cs.over:
				return nil
			default:
			}
			if len(d.ptn.args) < 2 {
				//定个标准,除了循环模式外,不传参数默认执行一次
				d.run()
				//无效信息,给我重新传参
				d.err = fmt.Errorf("error tick param ! you need to put a time.Duration type in second empty,put frequency int in third empty")
				return d.err
			} else {
				t, ok := d.ptn.args[0].(time.Duration)
				n, ok1 := d.ptn.args[1].(int)
				if ok && ok1 {
					for i := 0; i < n; i++ {
						d.run()
						<-d.cs.returns
						if i != n-1 {
							//间隔睡眠
							time.Sleep(t)
						}
					}
				} else {
					//定个标准,除了循环模式外,参数错误默认执行一次
					d.run()
					//无效信息,给我重新传参
					d.err = fmt.Errorf("error tick param ! you need to put a time.Duration type in second empty,put frequency int in third empty")
					return d.err
				}
			}
		}

	} else if params != nil && d.typ {
		//激活异步委托
		d.goRun = true
		go func() {
			//确定执行模式
			switch d.ptn.carryPattern {
			case Normal:
				select {
				case <-d.cs.over:
					return
				default:
				}
				d.run()
			case Cycle:
				//先确定ptn中的参数,
				//这里有个短路或,只要args长度为0,就不会再执行后面的语句了,就不会发生越界访问
				if len(d.ptn.args) == 0 || d.ptn.args[0] == -1 {

					for {
						select {
						case <-d.cs.over:
							return
						default:
						}
						d.run()
						//这里要做下清空管道操作,防止死锁
						<-d.cs.returns
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

						select {
						case <-d.cs.over:
							return
						default:
						}

						if i != 0 {
							<-d.cs.returns
						}
						d.run()
						//这里要做下清空管道操作,防止死锁
					}
				}

			case TimeOut:
				select {
				case <-d.cs.over:
					return
				default:
				}
				//没有传参或者无效时间参数则模式无效,正常执行
				if len(d.ptn.args) == 0 {
					d.run()
					d.err = fmt.Errorf("error time param ! you need to put a time.Duration type in second empty")
					//异步委托在返回前要写入通知chan

					//把错误写进去
					tmp := <-d.cs.returns
					tmp.err <- d.err
					d.cs.returns <- tmp

					d.cs.signal <- 1
					return
				}
				t, ok := d.ptn.args[0].(time.Duration)
				if ok {
					time.Sleep(t)
				} else {
					d.err = fmt.Errorf("error time param ! you need to put a time.Duration type in second empty")

				}
				d.run()
				//把错误写进去
				tmp := <-d.cs.returns
				tmp.err <- d.err
				d.cs.returns <- tmp

			case Tick: //间隔循环执行,第一个参数是间隔时间,第二个参数是次数
				select {
				case <-d.cs.over:
					return
				default:
				}
				if len(d.ptn.args) < 2 {
					//定个标准,除了循环模式外,不传参数默认执行一次
					d.run()
					//无效信息,给我重新传参
					d.err = fmt.Errorf("error tick param ! you need to put a time.Duration type in second empty,put frequency int in third empty")
					//异步委托在返回前要写入通知chan
					//把错误写进去
					tmp := <-d.cs.returns
					tmp.err <- d.err
					d.cs.returns <- tmp

					d.cs.signal <- 1
					return
				} else {
					t, ok := d.ptn.args[0].(time.Duration)
					n, ok1 := d.ptn.args[1].(int)
					if ok && ok1 {
						for i := 0; i < n; i++ {
							d.run()
							<-d.cs.returns
							if i != n-1 {
								//间隔睡眠
								time.Sleep(t)
							}
						}
					} else {
						//定个标准,除了循环模式外,参数错误默认执行一次
						d.run()
						//无效信息,给我重新传参
						d.err = fmt.Errorf("error tick param ! you need to put a time.Duration type in second empty,put frequency int in third empty")
						//异步委托在返回前要写入通知chan
						//把错误写进去
						tmp := <-d.cs.returns
						tmp.err <- d.err
						d.cs.returns <- tmp

						d.cs.signal <- 1
						return
					}
				}
			}

			d.cs.signal <- 1
		}()
	}
	return d.err
}

// GetReturns 获取返回值管理单元,自带同步阻塞,会等到委托执行完毕
func (d *delegator) GetReturns() (Returner, error) {
	//先判断是否执行完毕,没执行完毕会等待执行完毕
	if d.err != nil {
		return nil, d.err
	}
	//等全部执行完了才能拿返回值
	if d.goRun {
		d.Wait()
	}
	tmp := <-d.cs.returns
	return &tmp, nil
}

func (d *delegator) back() *delegator {
	return d
}

// Join 连接另一个委托,叠加两个委托的函数队列
func (d *delegator) Join(d2 Delegator) Delegator {
	if d.err != nil {
		return d
	}
	//要同步一下等待组的代办数和函数名
	d.fs = append(d.fs, d2.back().fs...)
	d.sleeper = append(d.sleeper, d2.back().sleeper...)
	d.names = append(d.names, d2.back().names...)

	return d
}

// SetTime 设置指定函数的睡眠时间,index参数为负数或超出委托成员长度则全部设置等长睡眠
func (d *delegator) SetTime(t time.Duration, index int) Delegator {
	if d.err != nil {
		return d
	}
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
	if d.err != nil {
		return d
	}
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

func (r *returner) BackError() error {
	return <-r.err
}
