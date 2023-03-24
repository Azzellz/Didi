package Delegator

import (
	"errors"
	"fmt"
)

type Proxy interface {
	back() *proxy
	Register(d Delegator, name string) error
	Execute(name string, args ...interface{}) error
	Flow(index int, args ...interface{}) error
	Delete(name string) error
	Get(name string) (Delegator, error)
	BackReturn(args ...interface{}) (Returner, error)
}

type proxy struct {
	dMap       map[string]delegatorUnit //非线性
	dArr       []delegatorUnit          //线性
	memberNums int
}

type delegatorUnit struct {
	id   int
	name string
	d    Delegator
	r    Returner
}

func New_() Proxy {
	return &proxy{dMap: make(map[string]delegatorUnit), dArr: make([]delegatorUnit, 0), memberNums: 0}
}

func (p *proxy) back() *proxy {
	return p
}

func (p *proxy) Register(d Delegator, name string) error {
	if _, ok := p.dMap[name]; ok {
		return errors.New("already proxy's name")
	} else {
		tmpD := delegatorUnit{d: d, name: name, id: p.memberNums}
		p.dMap[name] = tmpD
		p.dArr = append(p.dArr, tmpD)
		p.memberNums++
		return nil
	}
}

func (p *proxy) Get(name string) (Delegator, error) {
	if v, ok := p.dMap[name]; ok {
		return v.d, nil
	} else {
		return nil, errors.New("can't found the target")
	}
}

func (p *proxy) Delete(name string) error {
	flag := false
	pos := 0
	for i, v := range p.dArr {
		if v.name == name {
			pos = i
			flag = true
		}
	}
	if !flag {
		return errors.New("can't found the target")
	}

	delete(p.dMap, name)
	p.dArr = append(p.dArr[:pos], p.dArr[:pos+1]...)
	p.memberNums--

	return nil
}

func (p *proxy) BackReturn(args ...interface{}) (Returner, error) {
	if len(args) == 0 {
		return nil, errors.New("wrong first param , it should be int or string")
	}

	//根据传入的第一个参数是名字还是索引来选择数据结构获取
	switch v := args[0].(type) {
	case int:
		returner, err := p.dArr[v].d.GetReturns()
		return returner, err
	case string:
		returner, err := p.dMap[v].d.GetReturns()
		return returner, err
	}

	return nil, errors.New("wrong first param , it should be int or string")
}

// Execute 按map执行
func (p *proxy) Execute(name string, args ...interface{}) error {
	if _, ok := p.dMap[name]; !ok {
		return errors.New("proxy dont has target")
	} else {
		if len(args) == 0 {
			return p.dMap[name].d.Run()
		} else {
			return p.dMap[name].d.Run(args[0])
		}
	}
}

// Flow 流式执行
func (p *proxy) Flow(index int, args ...interface{}) error {

	if index > p.memberNums-1 {
		return errors.New("beyond the member nums")
	}

	if len(args) == 0 {
		if index < 0 {
			for _, v := range p.dArr {
				err := v.d.Run()
				if err != nil {
					fmt.Printf("%d号委托异常", v.id)
					return err
				}
			}
		} else {
			err := p.dArr[index].d.Run()
			if err != nil {
				fmt.Printf("%d号委托异常", index)
				return err
			}
		}
	} else {
		if index < 0 {
			for _, v := range p.dArr {
				err := v.d.Run(args[0])
				if err != nil {
					fmt.Printf("%d号委托异常", v.id)
					return err
				}
			}
		} else {
			err := p.dArr[index].d.Run(args[0])
			if err != nil {
				fmt.Printf("%d号委托异常", index)
				return err
			}
		}
	}
	return nil
}
