package main

import (
	"fmt"
	"time"
	"halftwo/mangos/xic"
)

type _DemoServant struct {
	xic.DefaultServant
	adapter xic.Adapter
}

func newServant(adapter xic.Adapter) *_DemoServant {
	setting := adapter.Engine().Setting()
	name := setting.Get("demo.name")

	srv := &_DemoServant{adapter:adapter}
	if name != "" {
		adapter.AddServant(name, srv)
	}
	return srv
}

func (srv *_DemoServant) Xic_echo(cur xic.Current, in xic.Arguments, out *xic.Arguments) error {
	out.CopyFrom(in)
	return nil
}

type _TimeInArgs struct {
	Time int64 `vbs:"time,omitempty"`
}

type _Times struct {
	Utc string `vbs:"utc"`
	Local string `vbs:"local"`
}

type _TimeOutArgs struct {
	Con string `vbs:"con"`
	Time int64 `vbs:"time"`
	Strftime _Times `vbs:"strftime"`
}

func (srv *_DemoServant) Xic_time(cur xic.Current, in _TimeInArgs, out *_TimeOutArgs) error {
	var t time.Time
	if in.Time == 0 {
		t = time.Now()
	} else {
		t = time.Unix(in.Time, 0)
	}
	const format = "2006-01-02T03:04:05-07:00 Mon"
	out.Con = cur.Con().String()
	out.Time = t.Unix()
	out.Strftime.Utc = t.UTC().Format(format)
	out.Strftime.Local = t.Format(format)
	return nil
}


func run(engine xic.Engine, args []string) error {
	adapter, err := engine.CreateAdapter("")
	if err != nil {
		return err
	}

	servant := newServant(adapter)
	_, err = adapter.AddServant("Demo", servant)
	if err != nil {
		fmt.Println("ERR", err)
	}
	adapter.Activate()
	engine.WaitForShutdown()
	return nil
}

func main() {
	xic.Start(run)
}

